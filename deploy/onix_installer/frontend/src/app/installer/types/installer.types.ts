/**
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// installer/frontend copy/src/app/installer/types/installer.types.ts

export interface GcpConfiguration {
  projectId: string;
  region: string;
}

export type InstallerGoal = 'create_new_open_network' | 'join_existing_network';

export type DeploymentGoal = {
  bap: boolean;
  bpp: boolean;
  gateway: boolean;
  registry: boolean;
  all?: boolean; // If all components are selected
};

export interface InfraOutputDetail {
  value: string;
}

export interface InfraDetails {
  cluster_name?: InfraOutputDetail;
  redis_instance_ip?: InfraOutputDetail;
  topic_name?: InfraOutputDetail;
  external_ip?: InfraOutputDetail;
  app_external_ip?: InfraOutputDetail;
  global_ip_address?: InfraOutputDetail;
  registry_url?: InfraOutputDetail;
  [key: string]: InfraOutputDetail | any;
}

export interface SubdomainConfig {
  component: 'registry' | 'gateway' | 'subscriber' | 'bap' | 'bpp' | 'adapter' | 'common' | 'registry_admin';
  domainType: 'google_domain' | 'other_domain';
  googleDomain?: {
    domain: string;
    zone: string;
  };
  customDomain?: {
    globalIp: string;
  };
  subdomainName: string;
}

// Existing DockerImageConfig
export interface DockerImageConfig {
  component: 'registry' | 'registry_admin' | 'gateway' | 'subscriber' | 'adapter';
  imageUrl: string;
}

export interface AppSpecificConfig {
  component: 'image' | 'registry' | 'gateway' | 'subscriber' | 'adapter';
  configs: { [key: string]: any };
  l2DomainSchemaFile?: File;
}

export interface HealthCheckStatus {
  serviceName: string;
  status: 'pending' | 'checking' | 'success' | 'failed';
  message?: string;
}

export type DeploymentStatus = 'pending' | 'in-progress' | 'completed' | 'failed';


export type DomainProvider = 'google_domain' | 'other_domain';

export interface DomainConfig {
  domainType: DomainProvider;
  baseDomain: string;
  dnsZone: string;
}

export type ComponentDomainKey = 'registry' | 'registry_admin' | 'gateway' | 'adapter' | 'subscriber';

export interface ComponentSubdomainPrefix {
  component: ComponentDomainKey;
  subdomainPrefix: string;
}

export type DeploymentSize = 'small' | 'medium' | 'large';

// --- NEW INTERFACES FOR APP DEPLOYMENT FORMS ---

export interface AppDeployImageConfig {
  registryImageUrl: string;
  registryAdminImageUrl: string;
  gatewayImageUrl: string;
  adapterImageUrl: string;
  subscriptionImageUrl: string;
}

export interface AppDeployRegistryConfig {
  registryUrl: string;
  registryKeyId: string;
  registrySubscriberId: string;
  enableAutoApprover: boolean;
}

export interface AppDeployGatewayConfig {
  gatewaySubscriptionId: string;
}

export interface AppDeployAdapterConfig {
  enableSchemaValidation: boolean;
}

export interface InstallerState {
  currentStepIndex: number;
  highestStepReached: number;
  installerGoal: 'create_new_open_network' | 'join_existing_network' | null;
  deploymentGoal: DeploymentGoal;
  prerequisitesMet: boolean;
  gcpConfiguration: GcpConfiguration | null;
  appName: string;
  deploymentSize: DeploymentSize;
  infraDetails: InfraDetails | null;
  appExternalIp: string | null;
  globalDomainConfig: DomainConfig | null;
  componentSubdomainPrefixes: ComponentSubdomainPrefix[];
  subdomainConfigs: SubdomainConfig[];

  dockerImageConfigs: DockerImageConfig[];
  appSpecificConfigs: AppSpecificConfig[]; // Keep this if used elsewhere, but new interfaces are more specific for app deploy forms
  healthCheckStatuses: HealthCheckStatus[];
  deploymentStatus: DeploymentStatus;
  appDeploymentStatus: DeploymentStatus;
  deploymentLogs: string[];

  deployedServiceUrls: { [key: string]: string };
  servicesDeployed: string[]; // Add this line
  logsExplorerUrls: { [key: string]: string }

  appDeployImageConfig: AppDeployImageConfig | null;
  appDeployRegistryConfig: AppDeployRegistryConfig | null;
  appDeployGatewayConfig: AppDeployGatewayConfig | null;
  appDeployAdapterConfig: AppDeployAdapterConfig | null;
}


export interface WelcomeFormValue {}
export interface GoalFormValue {
  installerGoal: InstallerGoal;
  deploymentGoal: DeploymentGoal;
}
export interface PrerequisitesFormValue {
  prerequisitesChecks: { [key: string]: boolean };
}
export interface GcpConnectionFormValue {
  projectId: string;
  region: string;
}
export interface DeployInfraFormValue {
  appName: string;
  deploymentSize: DeploymentSize;
}


export interface DeployAppFormValue {
  dockerImages: DockerImageConfig[];
  appSpecificConfigs: AppSpecificConfig[];
}
export interface SummaryFormValue {}

export interface InfraDeploymentRequestPayload {
  project_id: string;
  region: string;
  app_name: string;
  type: DeploymentSize;
  components: {
    gateway: boolean;
    registry: boolean;
    bap: boolean;
    bpp: boolean;
  };
}

export interface Base64File {
  name: string;
  content: string;
}

export interface AdapterConfig {
  enable_schema_validation?: boolean;
  l2_schema_files_content?: { [fileName: string]: string };
  bap_txn_receiver_file_content?: string;
  bap_txn_caller_file_content?: string;
  bpp_txn_receiver_file_content?: string;
  bpp_txn_caller_file_content?: string;
}

export interface BackendAppDeploymentRequest {
  components: { [key: string]: boolean };
  domain_names: { [key: string]: string };
  image_urls: { [key: string]: string };
  registry_url: string;
  app_name: string;
  adapter_config?: AdapterConfig;
  registry_config: {
    subscriber_id: string;
    key_id: string;
    enable_auto_approver?: boolean;
  };
  gateway_config?: { subscriber_id: string };
  domain_config: DomainConfig;
}

export interface HealthCheckItem {
  name:string;
  url: string;
  status: 'pending' | 'success' | 'failed' | null;
}


export interface HealthCheckPayload {
  service_urls_to_check: { [key: string]: string };
}
