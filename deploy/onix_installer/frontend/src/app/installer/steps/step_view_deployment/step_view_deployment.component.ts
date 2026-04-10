/**
 * Copyright 2026 Google LLC
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

import {Clipboard, ClipboardModule} from '@angular/cdk/clipboard';
import {CommonModule} from '@angular/common';
import {ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatIconModule} from '@angular/material/icon';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatTooltipModule} from '@angular/material/tooltip';  // Added Tooltip
import {Router} from '@angular/router';
import {EMPTY, Subject, Subscription} from 'rxjs';
import {catchError, finalize, takeUntil} from 'rxjs/operators';
import {windowOpen} from 'safevalues/dom';

// Services
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {removeEmptyValues} from '../../../shared/utils';
// Types
import {BackendAppDeploymentRequest} from '../../types/installer.types';


function isPayloadEqual(a: any, b: any): boolean {
  console.log('Config 1', a);
  console.log('Config 2', b);
  return JSON.stringify(a) === JSON.stringify(b);
}

@Component({
  selector: 'app-step-view-deployment',
  standalone: true,
  imports: [
    CommonModule, MatButtonModule, MatIconModule, MatProgressSpinnerModule,
    MatTooltipModule, ClipboardModule
    // KeyValuePipe is part of CommonModule
  ],
  templateUrl: './step_view_deployment.component.html',  // Ensure this matches
                                                         // your file name
  styleUrls: ['./step_view_deployment.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepViewDeployment implements OnInit, OnDestroy {
  // UI State Flags
  appDeploymentLogs: string[] = [];
  deploymentStatus: 'pending'|'in-progress'|'completed'|'failed' = 'pending';

  // Data for the Success View
  serviceUrls: {[key: string]: string} = {};
  appExternalIp: string|null = null;
  servicesDeployed: string[] = [];
  logsExplorerUrls: {[key: string]: string} = {};

  // Visibility Flags (Needed for HTML *ngIfs)
  showGatewayTab: boolean =
      false;  // Used to determine if Gateway logs should be shown
  showAdapterLogButton: boolean = false;

  // Cleanup
  private appWsSubscription!: Subscription;
  private unsubscribe$ = new Subject<void>();

  // Auto-scroll reference
  @ViewChild('appLogContainer') private appLogContainer!: ElementRef;

  constructor(
      private installerService: InstallerStateService,
      private webSocketService: WebSocketService,
      private cdr: ChangeDetectorRef,
      private router: Router,
      private clipboard: Clipboard,
  ) {}

  ngOnInit(): void {
    // 1. Initialize UI flags based on the previous config step
    const state = this.installerService.getCurrentState();
    const goal = state.deploymentGoal;

    const prevState = state.appDeploymentStatus;

    const currentPayload = this.generatePayload(state);
    const lastPayload = state.lastDeployedAppPayload;

    const isPayloadChanged =
        !isPayloadEqual(currentPayload, lastPayload) || state.isConfigChanged;
    const prevSuccess = state.appDeploymentStatus === 'completed';
    const prevFailed = state.appDeploymentStatus === 'failed';


    // Determine visibility for success output sections
    this.showGatewayTab = goal.all || goal.gateway;

    if (isPayloadChanged) {
      if (state.isConfigChanged)
        console.log(
            'Configuration changed since last deployment. Redeploying...');
      state.isConfigChanged = false;
      this.onDeploy();
    } else if (prevSuccess) {
      console.log(
          'Configuration unchanged and previously successful. Restoring view.');
      this.deploymentStatus = 'completed';
      this.restoreCompletedState(state);
    } else {
      // Config is same, but previous attempt failed or is pending -> Retry
      console.log('Retrying deployment...');
      this.onDeploy();
    }
  }

  ngOnDestroy(): void {
    if (this.appWsSubscription) {
      this.appWsSubscription.unsubscribe();
    }
    this.webSocketService.closeConnection();
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

  private restoreCompletedState(state: any): void {
    this.serviceUrls = state.deployedServiceUrls || {};
    this.servicesDeployed = state.servicesDeployed || [];
    this.logsExplorerUrls = state.logsExplorerUrls || {};
    this.appExternalIp = state.appExternalIp || null;

    // Logic to show/hide "Logs" button for adapter
    this.showAdapterLogButton =
        Object.keys(this.serviceUrls).some(key => key.startsWith('adapter_'));

    // Trigger change detection to update UI
    this.cdr.detectChanges();
  }

  public async onDeploy(): Promise<void> {
    const state = this.installerService.getCurrentState();


    const finalPayload = this.generatePayload(state);

    this.installerService.updateState({lastDeployedAppPayload: finalPayload});

    // UI Update
    this.deploymentStatus = 'in-progress';
    this.installerService.updateAppDeploymentStatus('in-progress');
    this.appDeploymentLogs = [];
    this.cdr.detectChanges();

    const goal = state.deploymentGoal;
    const deployAdapter = goal.all || goal.bap || goal.bpp;

    const wsUrl = `ws://localhost:8000/ws/deployApp`;

    // Connect to WebSocket
    this.appWsSubscription =
        this.webSocketService.connect(wsUrl)
            .pipe(
                takeUntil(this.unsubscribe$), catchError(error => {
                  this.handleDeploymentError(
                      `WebSocket connection error: ${error.message}`);
                  return EMPTY;
                }),
                finalize(() => {
                  // Safety check if socket closes unexpectedly
                  const currentState = this.installerService.getCurrentState();
                  if (currentState.appDeploymentStatus === 'in-progress') {
                    this.handleDeploymentError(
                        'Deployment failed: The connection was lost unexpectedly.');
                  }
                  this.webSocketService.closeConnection();
                }))
            .subscribe(
                {next: (message) => this.handleWebSocketMessage(message)});

    this.webSocketService.sendMessage(finalPayload);
  }

  private generatePayload(state: any): any {
    const goal = state.deploymentGoal;
    const deployAdapter = goal.all || goal.bap || goal.bpp;

    const potentialDomainNames = {
      registry:
          state.subdomainConfigs?.find((c: any) => c.component === 'registry')
              ?.subdomainName,
      registry_admin: state.subdomainConfigs
                          ?.find((c: any) => c.component === 'registry_admin')
                          ?.subdomainName,
      subscriber:
          state.subdomainConfigs?.find((c: any) => c.component === 'subscriber')
              ?.subdomainName,
      gateway:
          state.subdomainConfigs?.find((c: any) => c.component === 'gateway')
              ?.subdomainName,
      adapter:
          state.subdomainConfigs?.find((c: any) => c.component === 'adapter')
              ?.subdomainName
    };

    const potentialImageUrls = {
      registry: state.appDeployImageConfig?.registryImageUrl,
      registry_admin: state.appDeployImageConfig?.registryAdminImageUrl,
      subscriber: state.appDeployImageConfig?.subscriptionImageUrl,
      gateway: state.appDeployImageConfig?.gatewayImageUrl,
      adapter: state.appDeployImageConfig?.adapterImageUrl
    };

    const payload: BackendAppDeploymentRequest = {
      app_name: state.appName,
      components: {
        adapter: deployAdapter,
        gateway: goal.all || goal.gateway,
        registry: goal.all || goal.registry,
        bap: goal.all || goal.bap,
        bpp: goal.all || goal.bpp
      },
      domain_names: removeEmptyValues(potentialDomainNames),
      image_urls: removeEmptyValues(potentialImageUrls),
      registry_url: state.appDeployRegistryConfig?.registryUrl || '',
      registry_config: {
        subscriber_id:
            state.appDeployRegistryConfig?.registrySubscriberId || '',
        key_id: state.appDeployRegistryConfig?.registryKeyId || '',
        enable_auto_approver:
            state.appDeployRegistryConfig?.enableAutoApprover || false
      },
      domain_config: {
        domainType: state.globalDomainConfig?.domainType || 'other_domain',
        baseDomain: state.globalDomainConfig?.baseDomain || '',
        dnsZone: state.globalDomainConfig?.dnsZone || ''
      }
    };

    if (goal.all || goal.gateway) {
      payload.gateway_config = {
        subscriber_id: state.appDeployGatewayConfig?.gatewaySubscriptionId || ''
      };
    }

    if (deployAdapter) {
      payload.adapter_config = {
        enable_schema_validation:
            state.appDeployAdapterConfig?.enableSchemaValidation || false
      };
    }

    const securityConfig = state.appDeploySecurityConfig;
    if (securityConfig) {
      payload.security_config = {
        enable_inbound_auth: securityConfig.enableInBoundAuth || false,
        issuer_url: securityConfig.enableInBoundAuth ?
            (securityConfig.issuerUrl || '') :
            '',
        idclaim: securityConfig.enableInBoundAuth ?
            (securityConfig.idclaim || '') :
            '',
        allowed_values: securityConfig.enableInBoundAuth &&
                securityConfig.allowedValues ?
            securityConfig.allowedValues.split(',')
                .map((val: string) => val.trim())
                .filter((val: string) => val.length > 0) :
            [],
        jwks_content: securityConfig.enableInBoundAuth ?
            (securityConfig.jwksFileContent || '') :
            '',
        enable_outbound_auth: securityConfig.enableOutBoundAuth || false,
        aud_overrides: securityConfig.enableOutBoundAuth ?
            (securityConfig.audOverrides || '') :
            ''
      };
    }

    return payload;
  }

  private handleWebSocketMessage(message: any): void {
    let parsed: any;
    try {
      parsed = typeof message === 'string' ? JSON.parse(message) : message;
    } catch (e) {
      this.appDeploymentLogs.push(String(message));
      this.scrollToBottom();  // Auto-scroll
      this.cdr.detectChanges();
      return;
    }

    const {type, message: msgContent, data} = parsed;

    switch (type) {
      case 'log':
        this.appDeploymentLogs.push(msgContent);
        break;

      case 'success':
        this.deploymentStatus = 'completed';
        this.installerService.updateAppDeploymentStatus('completed');
        this.appDeploymentLogs.push(
            'Application Deployment Completed Successfully!');

        if (data) {
          // Save results to state
          this.installerService.updateState({
            deployedServiceUrls: data.service_urls || {},
            servicesDeployed: data.services_deployed || [],
            logsExplorerUrls: data.logs_explorer_urls || {},
            appExternalIp: data.app_external_ip || null,
          });

          // Update local variables for the success UI
          this.serviceUrls = data.service_urls || {};
          this.servicesDeployed = data.services_deployed || [];
          this.logsExplorerUrls = data.logs_explorer_urls || {};
          this.appExternalIp = data.app_external_ip || null;

          // Logic to show/hide "Logs" button for adapter
          this.showAdapterLogButton =
              Object.keys(this.serviceUrls)
                  .some(key => key.startsWith('adapter_'));
        }
        break;

      case 'error':
        this.handleDeploymentError(msgContent);
        break;
    }

    this.cdr.detectChanges();
    this.scrollToBottom();
  }

  private handleDeploymentError(error: string) {
    this.deploymentStatus = 'failed';
    this.installerService.updateAppDeploymentStatus('failed');
    this.appDeploymentLogs.push(`${
        error}`);  // Cleaned up "Error:" prefix to match old style if desired
    this.cdr.detectChanges();
    this.scrollToBottom();
  }

  // --- UI Helper Methods ---

  private scrollToBottom(): void {
    try {
      if (this.appLogContainer && this.appLogContainer.nativeElement) {
        this.appLogContainer.nativeElement.scrollTop =
            this.appLogContainer.nativeElement.scrollHeight;
      }
    } catch (err) {
      console.warn('Scroll to bottom failed', err);
    }
  }

  openUrl(url: string|null): void {
    if (url) {
      windowOpen(window, url, '_blank');
    }
  }

  copyToClipboard(text: string|null): void {
    if (text) {
      this.clipboard.copy(text!);
      console.log('Copied to clipboard:', text);
    }
  }

  onRetry(): void {
    this.router.navigate(['installer', 'domain-configuration']);

    // this.appDeploymentLogs = [];
    // this.onDeploy();
  }

  onContinueToNextStep(): void {
    this.router.navigate(['installer', 'health-checks']);
  }
}