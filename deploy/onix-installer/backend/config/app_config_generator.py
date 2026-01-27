# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
import os
from typing import Dict, List

import urllib

from core.models import AppDeploymentRequest, ConfigGenerationRequest
from core.constants import TERRAFORM_DIRECTORY, TEMPLATE_DIRECTORY, GENERATED_CONFIGS_DIR
from core import utils

logger = logging.getLogger(__name__)

# Filenames of Jinja2 templates.
ADAPTER_CONFIG_TEMPLATE_NAME = "adapter.yaml.j2"
REGISTRY_CONFIG_TEMPLATE_NAME = "registry.yaml.j2"
GATEWAY_CONFIG_TEMPLATE_NAME = "gateway.yaml.j2"
SUBSCRIBER_CONFIG_TEMPLATE_NAME = "subscriber.yaml.j2"
REGISTRY_ADMIN_CONFIG_TEMPLATE_NAME = "registry-admin.yaml.j2"
TFVARS_TEMPLATE_NAME = "p2.tfvars.j2"

def _should_deploy_adapter(components: dict) -> bool:
    """
    Determines if the adapter should be deployed based on the 'bap' or 'bpp' components.
    """
    return components.get("bap", False) or components.get("bpp", False)


def should_deploy_subscriber(components: dict) -> bool:
    """
    Determines if the subscriber should be deployed based on the 'bap', 'bpp', or 'gateway' components.
    """
    return components.get("bap", False) or components.get("bpp", False) or components.get("gateway", False)


def _load_infrastructure_outputs(terraform_outputs_dir: str) -> dict:
    """
    Loads and returns infrastructure outputs from the 'outputs.json' file.
    """
    outputs_json_path = os.path.join(terraform_outputs_dir, "outputs.json")
    logger.info(f"Loading infrastructure outputs from {outputs_json_path}")
    try:
        raw_outputs = utils.read_json_file(outputs_json_path)
        infra_output_values = {k: v.get("value") for k, v in raw_outputs.items()}
        logger.info("Infrastructure outputs loaded successfully.")
        return infra_output_values
    except FileNotFoundError as e:
        logger.error(f"Infrastructure outputs file not found: {e}")
        raise
    except ValueError as e:
        logger.error(f"Error decoding infrastructure outputs JSON: {e}")
        raise
    except Exception as e:
        logger.exception(f"An unexpected error occurred while loading infrastructure outputs: {e}")
        raise


def _generate_file_from_template(
    template_source_dir: str,
    template_j2_filename: str,
    output_dir: str,
    context: dict,
    skip_if_exists: bool = True
):
    """
    Helper function to render a Jinja2 template and write the content to a file.
    """
    output_filename = template_j2_filename.replace('.j2', '')
    output_path = os.path.join(output_dir, output_filename)

    if os.path.exists(output_path) and skip_if_exists:
        logger.info(f"Skipping generation of '{output_path}' as it already exists.")
        return

    logger.info(f"Processing template: '{template_j2_filename}' -> '{output_path}'...")
    try:
        rendered_content = utils.render_jinja_template(
            template_dir=template_source_dir,
            template_name=template_j2_filename,
            context=context
        )
        utils.write_file_content(output_path, rendered_content)
        logger.debug(f"Generated successfully: {output_path}")
    except (FileNotFoundError, RuntimeError, IOError) as e:
        logger.error(f"Failed to generate '{output_filename}': {e}", exc_info=True)
        raise


def generate_app_configs(request: ConfigGenerationRequest) -> None:
    """Generates application configuration YAML files based on the request.
    Generates adapter.yaml, registry.yaml, etc., by loading infrastructure outputs
    and rendering the corresponding Jinja2 templates.
    Args:
        request: A ConfigGenerationRequest containing configuration parameters.
    Raises:
        FileNotFoundError: If infrastructure outputs or templates are missing.
        ValueError: If output data is invalid.
        RuntimeError: If generation logic fails.
    """
    logger.info("Starting Application Configuration YAML Generation ")

    try:
        # Loading infrastructure outputs.
        infra_output_values = _load_infrastructure_outputs(TERRAFORM_DIRECTORY)
        template_context = _prepare_config_generation_context(request, infra_output_values)

        os.makedirs(GENERATED_CONFIGS_DIR, exist_ok=True)

        templates_to_generate = set()

        # Determining which templates to generate based on components.
        if _should_deploy_adapter(request.components):
            templates_to_generate.add(ADAPTER_CONFIG_TEMPLATE_NAME)
            logger.debug("Adapter deployment requested. Adding adapter template.")
        if request.components.get("gateway", False):
            templates_to_generate.add(GATEWAY_CONFIG_TEMPLATE_NAME)
            logger.debug("Gateway deployment requested. Adding gateway template.")
        if should_deploy_subscriber(request.components):
            templates_to_generate.add(SUBSCRIBER_CONFIG_TEMPLATE_NAME)
            logger.debug("Subscriber deployment requested. Adding subscriber template.")
        if request.components.get("registry", False):
            templates_to_generate.add(REGISTRY_CONFIG_TEMPLATE_NAME)
            templates_to_generate.add(REGISTRY_ADMIN_CONFIG_TEMPLATE_NAME)
            logger.debug("Registry deployment requested. Adding registry and registry-admin templates.")

        logger.info(f"Templates selected for generation: {list(templates_to_generate)}")
        template_source_dir = os.path.join(TEMPLATE_DIRECTORY, 'configs')
        
        # Loop through templates and generate files.
        for template_j2_filename in templates_to_generate:
            _generate_file_from_template(
                template_source_dir=template_source_dir,
                template_j2_filename=template_j2_filename,
                output_dir=GENERATED_CONFIGS_DIR,
                context=template_context
            )

    except (FileNotFoundError, ValueError, IOError, RuntimeError) as e:
        logger.critical(f"Critical Error during Application Configuration YAML Generation: {e}", exc_info=True)
        raise

    logger.info("Application Config YAML files generation completed")


def generate_p2_tfvars(request: AppDeploymentRequest) -> None:
    """Generates the p2.tfvars file required for Phase 2 Terraform (HTTPS/SSL).
    Args:
        request: An AppDeploymentRequest containing domain and application details.
    Raises:
        Exception: If any error occurs during the template rendering or file writing process.
    """
    logger.info("Starting p2.tfvars Generation")
    try:
        infra_output_values = _load_infrastructure_outputs(TERRAFORM_DIRECTORY)
        template_context = _prepare_tfvars_context(request, infra_output_values)
        
        tf_vars_output_dir = os.path.join(TERRAFORM_DIRECTORY, 'phase2')
        tf_template_source_dir = os.path.join(TEMPLATE_DIRECTORY, 'tf_configs')
        
        template_j2_filename = TFVARS_TEMPLATE_NAME

        _generate_file_from_template(
                template_source_dir=tf_template_source_dir,
                template_j2_filename=template_j2_filename,
                output_dir=tf_vars_output_dir,
                context=template_context,
                skip_if_exists=False
            )
        logger.info("p2.tfvars generation completed")
    except Exception as e:
        logger.critical(f"Critical Error during p2.tfvars Generation: {e}", exc_info=True)
        raise


def get_deployment_environment_variables(app_deployment_request: AppDeploymentRequest, services_to_deploy: list[str]) -> dict[str, str]:
    """
    Prepares environment variables needed for the deploy-app.sh script based on the
    AppDeploymentRequest.
    """
    logger.info("Preparing environment variables for deploy-app.sh...")
    env_vars = {}

    env_vars["DEPLOY_SERVICES"] = ",".join(sorted(services_to_deploy))
    logger.debug(f"  DEPLOY_SERVICES environment variable set to: {env_vars['DEPLOY_SERVICES']}")

    # Add Domain Names to environment variables if provided.
    for key, domain in app_deployment_request.domain_names.items():
        env_var_name = f"{key.upper().replace('-', '_')}_DOMAIN"
        env_vars[env_var_name] = domain
        logger.debug(f"Setting ENV: {env_var_name}={domain}")

    # Add Image URLs to environment variables if provided.
    for key, url in app_deployment_request.image_urls.items():
        env_var_name = f"{key.upper().replace('-', '_')}_IMAGE_URL"
        env_vars[env_var_name] = url
        logger.debug(f"  Setting ENV: {env_var_name}={url}")

     # Check config file content for schema validation
    enable_schema_validation = False

    # Only check config if the adapter is actually being deployed
    if "adapter" in services_to_deploy:
        adapter_config_path = os.path.join(GENERATED_CONFIGS_DIR, "adapter.yaml")
        enable_schema_validation = _is_schema_validation_enabled(adapter_config_path)
    
    # Convert bool to string for the environment variable
    env_vars["ENABLE_SCHEMA_VALIDATION"] = str(enable_schema_validation).lower()
    logger.debug(f"  Setting ENV: ENABLE_SCHEMA_VALIDATION={env_vars['ENABLE_SCHEMA_VALIDATION']}")

    logger.info("Environment variables prepared for deploy-app.sh.")
    return env_vars


def _is_schema_validation_enabled(config_path: str) -> bool:
    """
    Parses the adapter configuration file to determine if schema validation
    is enabled in any of the modules.
    """
    if not os.path.exists(config_path):
        return False

    try:
        logger.debug(f"Inspecting {config_path} for schema validation settings...")
        adapter_data = utils.read_yaml_file(config_path)
        
        modules = adapter_data.get("modules", [])
        if isinstance(modules, list):
            for module in modules:
                if isinstance(module, dict):
                    # Safely traverse the nested dictionary structure
                    plugins = module.get("handler", {}).get("plugins", {})
                    if "schemaValidator" in plugins:
                        logger.info("Schema validation detected as ENABLED in adapter configuration.")
                        return True
                        
    except Exception as e:
        logger.warning(f"Failed to parse adapter config to verify schema validation status: {e}. Defaulting to false.")
    
    return False


def extract_final_urls(domain_names: Dict[str, str], services: List[str]) -> Dict[str, str]:

    logger.info("Extracting final URLs for services...")
    service_urls = {}
    logger.debug(f"Domain names provided: {domain_names}")
    for service_name in services:
        service_domain = domain_names.get(service_name)

        if not service_domain:
            logger.warning(f"Domain not found for service '{service_name}'. Skipping URL extraction for this service.")
            continue

        url = f"https://{service_domain}"
        if service_name == "adapter":
            service_urls[service_name] = url
            adapter_config_yaml_path = os.path.join(GENERATED_CONFIGS_DIR, "adapter.yaml")

            try:
                app_config_data = utils.read_yaml_file(adapter_config_yaml_path)

                if 'modules' in app_config_data and isinstance(app_config_data['modules'], list):
                    for module in app_config_data['modules']:
                        if isinstance(module, dict) and 'name' in module and 'path' in module:
                            module_name = module['name']
                            module_path = module['path']
                            if url:
                                combined_path = f"{url}/{module_path.lstrip('/')}"
                                service_urls[f"adapter_{module_name}"] = combined_path
                else:
                    logger.warning(f"'modules' key not found or not a list in '{adapter_config_yaml_path}'. Cannot extract adapter module paths.")

                logger.debug(f"Extracted adapter paths from '{adapter_config_yaml_path}': {service_urls}")

            except FileNotFoundError:
                logger.warning(f"Application config YAML for adapter not found at '{adapter_config_yaml_path}'. Skipping adapter module data extraction.")
            except ValueError as e:
                logger.error(f"Error parsing application config YAML from '{adapter_config_yaml_path}': {e}. Skipping adapter module data extraction.")
            except Exception as e:
                logger.error(f"An unexpected error occurred while extracting adapter module data from '{adapter_config_yaml_path}': {e}", exc_info=True)

        else:
            service_urls[service_name] = url
            logger.debug(f"Generated URL for {service_name}: {service_urls[service_name]}")

    return service_urls


def generate_logs_explorer_urls(service_names: List[str]) -> Dict[str, str]:
    """
    Generates Cloud Logs Explorer URLs for a given list of services,
    assuming container names follow the 'onix-{service_name}' pattern.
    It loads necessary infrastructure details from outputs.json.
    """

    logger.info("Generating Logs Explorer URLs for services...")
    logs_explorer_urls = {}
    try:
        infra_output_values = _load_infrastructure_outputs(TERRAFORM_DIRECTORY)
        project_id = infra_output_values.get("project_id")
        cluster_name = infra_output_values.get("cluster_name")
        cluster_region = infra_output_values.get("cluster_region")

        for service_name in service_names:
            container_name = f"onix-{service_name.replace('_', '-')}"
            log_query_parts = [
                f'resource.type="k8s_container"',
                f'resource.labels.cluster_name="{cluster_name}"',
                f'resource.labels.location="{cluster_region}"',
                f'resource.labels.container_name="{container_name}"'
            ]
            log_query = "\n".join(log_query_parts)

            # URL-encode the query string
            encoded_log_query = urllib.parse.quote(log_query)

            # Construct the full Logs Explorer URL
            logs_explorer_url = (
                f"https://console.cloud.google.com/logs/query;"
                f"query={encoded_log_query};"
                f"?project={project_id}"
            )
            logs_explorer_urls[service_name] = logs_explorer_url
            logger.debug(f"Generated Logs Explorer URL for {service_name}: {logs_explorer_url}")

    except Exception as e:
        logger.warning(f"An error occurred while generating Logs Explorer URLs: {e}", exc_info=True)

    logger.info("Generated Logs Explorer URLs.")
    return logs_explorer_urls


def _get_common_infra_context(infra_output_values: dict) -> dict:
    """
    Extracts and sanitizes common infrastructure values required by the templates.
    """
    iam_sa_suffix = ".gserviceaccount.com"
    
    # safely remove suffix if the value exists
    def clean_email(email):
        return email.removesuffix(iam_sa_suffix) if email else ""

    return {
        "project_id": infra_output_values.get("project_id"),
        "region": infra_output_values.get("cluster_region"),
        "cluster_region": infra_output_values.get("cluster_region"),
        "redis_instance_ip": infra_output_values.get("redis_instance_ip"),
        "onix_topic_name": infra_output_values.get("onix_topic_name"),
        "adapter_topic_name": infra_output_values.get("adapter_topic_name"),
        
        # Database Users
        "database_user_sa_email": clean_email(infra_output_values.get("database_user_sa_email")),
        "registry_admin_database_user_sa_email": clean_email(infra_output_values.get("registry_admin_database_user_sa_email")),
        
        # Database Connection Info
        "registry_database_name": infra_output_values.get("registry_database_name"),
        "registry_db_connection_name": infra_output_values.get("registry_db_connection_name"),
        
        # Global Networking
        "config_bucket_name": infra_output_values.get("config_bucket_name"),
        "url_map": infra_output_values.get("url_map", ""),
        "global_ip_address": infra_output_values.get("global_ip_address"),
    }


def _prepare_config_generation_context(request: ConfigGenerationRequest, infra_output_values: dict) -> dict:
    """
    Prepares the context dictionary specifically for generating App Configuration YAMLs.
    Maps ConfigGenerationRequest fields to the Jinja2 variables identified in the templates.
    """
    logger.debug("Preparing Jinja2 template context for App Configuration...")
    
    context = _get_common_infra_context(infra_output_values)

    registry_data = request.registry_config.model_dump()
    adapter_data = request.adapter_config.model_dump() if request.adapter_config else {}
    gateway_data = request.gateway_config.model_dump() if request.gateway_config else {}

    context.update({
        "deploy_bap": request.components.get("bap", False),
        "deploy_bpp": request.components.get("bpp", False),
        "enable_subscriber": should_deploy_subscriber(request.components),
        "enable_auto_approver": request.registry_config.enable_auto_approver,

        "registry": registry_data,
        "adapter": adapter_data,
        "gateway": gateway_data,

        "registry_url": str(request.registry_url) if request.registry_url else "",
    })
    
    return context


def _prepare_tfvars_context(request: AppDeploymentRequest, infra_output_values: dict) -> dict:
    """
    Prepares the context dictionary specifically for generating p2.tfvars.
    Maps AppDeploymentRequest fields and Infra Outputs to the variables in p2.tfvars.j2.
    """
    logger.debug("Preparing Jinja2 template context for Terraform Variables (p2.tfvars)...")

    context = _get_common_infra_context(infra_output_values)

    is_google_domain = (request.domain_config.domainType == "google_domain")
    
    context.update({
        "suffix": request.app_name, 
        "is_google_domain": is_google_domain,
        "dns_zone": request.domain_config.dnsZone,
        "domains": request.domain_names, 
        "domain_list": list(request.domain_names.values()),
        "enable_subscriber": should_deploy_subscriber(request.components),
        "enable_auto_approver": request.registry_config.enable_auto_approver,
    })

    return context