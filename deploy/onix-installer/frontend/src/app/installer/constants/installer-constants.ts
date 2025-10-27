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

export const InstallerConstants = {
  STEPS: [
    'Welcome',
    'Goal',
    'Prerequisites',
    'GCP Connection',
    'Deploy Infra',
    'Domain Configuration',
    'Deploy App',
    'Health Checks',
    'Subscribe'
  ],

  STEP_PATHS: [
    'welcome',
    'goal',
    'prerequisites',
    'gcp-connection',
    'deploy-infra',
    'domain-configuration',
    'deploy-app',
    'health-checks',
    'subscribe',
  ],


  // TODO : remove innner HTML from content
  PREREQUISITES_LIST: [
    'I have active access to a Google Cloud Platform (GCP) Project with <strong>Project Editor</strong> permissions.',
    'I have installed the Google Cloud SDK (gcloud) and authenticated my user account.',
    'I have obtained the <strong>Registry and Gateway URLs</strong> for an existing Beckn-compliant network.',
    'I have the <strong>L2 domain schema file</strong> ready for upload.',
    'I have a BAP/BPP backend application ready for testing.',
  ],
};