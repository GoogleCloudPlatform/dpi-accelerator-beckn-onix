/**
 *  Copyright 2026 Google LLC
 * 
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 * 
 *      http://www.apache.org/licenses/LICENSE-2.0
 * 
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */


import {CommonModule} from '@angular/common';
import {ChangeDetectorRef, Component, OnInit} from '@angular/core';
import {FormsModule} from '@angular/forms';
import {MatButton} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatSpinner} from '@angular/material/progress-spinner';
import {MatTooltip} from '@angular/material/tooltip';
import {Router} from '@angular/router';
import * as yaml from 'js-yaml';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';



interface ConfigFileItem {
  path: string;
  name: string;
  type: 'file'|'folder';
  parentFolder: string|null;
  isHidden: boolean;
  isExpanded: boolean;
}

@Component({
  selector: 'app-step-view-config',
  standalone: true,
  imports:
      [CommonModule, MatButton, MatIcon, MatTooltip, MatSpinner, FormsModule],
  templateUrl: './step_view_config.component.html',
  styleUrl: './step_view_config.component.css'
})
export class StepViewConfigComponent implements OnInit {
  validationError: string|null = null;
  isLoading = false;

  isEditing = false;
  currentFile: any = null;
  fileContent: string = '';
  isSaving = false;

  // Data
  files: ConfigFileItem[] = [];

  originalFileContent: string = '';

  // editingFile: { path: string, content: string } | null = null;

  constructor(
      private router: Router, private apiService: ApiService,
      private installerService: InstallerStateService,
      private cdr: ChangeDetectorRef) {}

  ngOnInit(): void {
    this.generateAndLoadConfigs();
  }

  generateAndLoadConfigs() {
    this.isLoading = true;
    // Step 1: Generate Configs using data from InstallerService
    const payload = this.installerService.getCurrentState();
    const securityConfig = payload.appDeploySecurityConfig;

    const hardcodedPayload = {
      'app_name': payload.appName,
      'components': {
        'bap': payload.deploymentGoal.bap,
        'bpp': payload.deploymentGoal.bpp,
        'registry': payload.deploymentGoal.registry,
        'gateway': payload.deploymentGoal.gateway
      },
      'registry_url': payload.appDeployRegistryConfig?.registryUrl,
      'registry_config': {
        'subscriber_id': payload.appDeployRegistryConfig?.registrySubscriberId,
        'key_id': payload.appDeployRegistryConfig?.registryKeyId,
        'enable_auto_approver':
            payload.appDeployRegistryConfig?.enableAutoApprover
      },
      'adapter_config': {
        'enable_schema_validation':
            payload.appDeployAdapterConfig?.enableSchemaValidation
      },
      'gateway_config': {
        'subscriber_id': payload.appDeployGatewayConfig?.gatewaySubscriptionId
      },

      'security_config': securityConfig ? {
        'enable_inbound_auth': securityConfig.enableInBoundAuth || false,
        'issuer_url': securityConfig.enableInBoundAuth ?
            (securityConfig.issuerUrl || '') :
            '',
        'jwks_content': securityConfig.enableInBoundAuth ?
            (securityConfig.jwksFileContent || '') :
            '',
        'enable_outbound_auth': securityConfig.enableOutBoundAuth || false,
        'aud_overrides': securityConfig.enableOutBoundAuth ?
            (securityConfig.audOverrides || '') :
            ''
      } :
                                          undefined
    };

    // console.log(hardcodedPayload);

    this.apiService.postConfigs(hardcodedPayload).subscribe({
      next: () => this.fetchFilePaths(),
      error: (err) => {
        console.error(err);
        this.isLoading = false;
      }
    });
  }

  fetchFilePaths() {
    this.apiService.getConfigPaths().subscribe(res => {
          const processedFiles: ConfigFileItem[] = [];

          res.files.forEach(path => {
            const parts = path.split('/');

            // If path is "Beckn/source.yaml", parts[0] is the folder
            if (parts.length > 1) {
              const folderName = parts[0];
              // Add folder if not already added
              if (!processedFiles.find(f => f.name === folderName)) {
                processedFiles.push({
                  path: folderName,
                  name: folderName,
                  type: 'folder',
                  parentFolder: null,
                  isHidden: false,
                  isExpanded: false
                });
              }
              // Add the file as a child
              processedFiles.push({
                path,
                name: parts[1],
                type: 'file',
                parentFolder: folderName,
                isHidden: true,
                isExpanded: false
              });
            } else {
              // Root level file
              processedFiles.push({
                path,
                name: path,
                type: 'file',
                parentFolder: null,
                isHidden: false,
                isExpanded: false
              });
            }
          });
          this.files = processedFiles;
          this.cdr.detectChanges();
        });
  }

  onProceedToDeploy() {
    this.router.navigate(['installer', 'view-deployment']);
  }

  toggleFolder(folder: any) {
    folder.isExpanded = !folder.isExpanded;
    // Toggle visibility for all files that belong to this folder
    this.files.forEach(f => {
      if (f.parentFolder === folder.name) {
        f.isHidden = !folder.isExpanded;
      }
    });
  }

  onEditFile(file: any) {
    this.currentFile = file;
    this.isLoading = true;
    this.apiService.getConfigData(file.path).subscribe({
      next: (res) => {
        this.fileContent = res.content;

            this.originalFileContent = res.content;
            this.isEditing = true;
            this.isLoading = false;
          },
          error: () => this.isLoading = false
        });
  }

  private findMissingValue(data: any, prefix: string = ''): string|null {
    // 1. If the value itself is null, we found an empty field
    if (data === null) {
      return prefix;
    }

    // 2. If it's an object, check its children
    if (typeof data === 'object' && data !== undefined) {
      for (const key in data) {
        if (Object.prototype.hasOwnProperty.call(data, key)) {
          const currentPath = prefix ? `${prefix}.${key}` : key;

          // Recursive call
          const missingKey = this.findMissingValue(data[key], currentPath);
          if (missingKey) {
            return missingKey;
          }
        }
      }
    }

    // 3. No missing values found
    return null;
  }

  onSave() {
    // 1. Clear previous errors
    this.validationError = null;
    let parsedData: any;

    // 2. Client-Side Validation
    try {
      // Attempt to load the YAML. If the syntax is invalid, this throws an
      // error. We don't strictly need the result, just checking if it throws.
      parsedData = yaml.load(this.fileContent);
    } catch (e: any) {
      // 3. Handle Invalid YAML
      console.error('YAML Validation Error:', e);

      // js-yaml errors are very descriptive (line numbers, reason).
      // We extract the message to show the user.
      this.validationError = `Invalid YAML: ${e.message}`;
      return;  // <--- STOP here. Do not call API.
    }

    const missingField = this.findMissingValue(parsedData);

    if (missingField) {
      this.validationError = `Configuration Error: The value for '${
          missingField}' cannot be empty.`;
      return;  // <--- STOP here
    }


    // 3. NEW: Check for changes before saving
    if (this.fileContent !== this.originalFileContent) {
      console.log('File content changed. Marking state as modified.');
      this.installerService.updateState({isConfigChanged: true});
    }


    this.isSaving = true;
    const payload = {path: this.currentFile.path, content: this.fileContent};

    this.apiService.updateConfigData(payload).subscribe({
      next: () => {
        this.isSaving = false;
        this.isEditing = false;
        // Optional: Show a success toast here
        this.cdr.detectChanges();
      },
      error: () => {
        this.isSaving = false;
        this.cdr.detectChanges();
      }
    });
  }

  onCancel() {
    this.isEditing = false;
    this.currentFile = null;
    this.fileContent = '';
  }

  // onDeploy() {
  //   this.router.navigate(['installer', 'health-checks'])
  // }

  onBack() {
    this.router.navigate(['installer', 'app-config'])
  }
}
