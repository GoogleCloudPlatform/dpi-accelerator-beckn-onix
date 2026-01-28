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

import { Injectable } from '@angular/core';
import {  BehaviorSubject, Observable, catchError, of  } from 'rxjs';
import {
  InstallerState,
  DeploymentGoal,
  GcpConfiguration,
  InfraDetails,
  SubdomainConfig,
  DockerImageConfig,
  AppSpecificConfig,
  HealthCheckStatus,
  DeploymentStatus,
  DomainConfig,
  ComponentSubdomainPrefix,
  DeploymentSize,
  AppDeployImageConfig,
  AppDeployRegistryConfig,
  AppDeployGatewayConfig,
  AppDeployAdapterConfig
} from '../../installer/types/installer.types';
import { ApiService } from './api.service';


@Injectable({
  providedIn: 'root',
})
export class InstallerStateService {
  private initialState: InstallerState = {
    currentStepIndex: 0,
    highestStepReached: 0,
    installerGoal: null,
    deploymentGoal: { bap: false, bpp: false, gateway: false, registry: false, all: false },
    prerequisitesMet: false,
    gcpConfiguration: null,
    appName: '',
    deploymentSize: 'small',
    infraDetails: null,
    appExternalIp: null,
    deployedServiceUrls: {},
    servicesDeployed: [],
    logsExplorerUrls: {},
    globalDomainConfig: null,
    componentSubdomainPrefixes: [],

    subdomainConfigs: [],
    dockerImageConfigs: [],
    appSpecificConfigs: [],
    healthCheckStatuses: [],
    deploymentStatus: 'pending',
    appDeploymentStatus: 'pending',
    deploymentLogs: [],

    appDeployImageConfig: {
      registryImageUrl: '',
      registryAdminImageUrl: '',
      gatewayImageUrl: '',
      adapterImageUrl: '',
      subscriptionImageUrl: ''
    },
    appDeployRegistryConfig: {
      registryUrl: '',
      registryKeyId: '',
      registrySubscriberId: '',
      enableAutoApprover: false
    },
    appDeployGatewayConfig: {
      gatewaySubscriptionId: ''
    },
    appDeployAdapterConfig: {
      enableSchemaValidation: false
    },
  };

  private isDeployingSubject = new BehaviorSubject<boolean>(false);
  isDeploying$ = this.isDeployingSubject.asObservable();

  private isStateLoadingSubject = new BehaviorSubject<boolean>(true);
  public isStateLoading$ = this.isStateLoadingSubject.asObservable();

  private installerStateSubject = new BehaviorSubject<InstallerState>(
    this.initialState
  );
  installerState$: Observable<InstallerState> = this.installerStateSubject.asObservable();


  // NOTE: REMOVE HTTP Calls state service Components should call ApiService and then update the state in InstallerStateService with the results.
  constructor(private apiService: ApiService) {
    this.loadStateFromBackend();
  }

   private loadStateFromBackend(): void {
    this.isStateLoadingSubject.next(true);
    this.apiService.getState().pipe(
      catchError(error => {
        console.error("Could not load state from backend, starting with initial state.", error);
        return of(null);
      })
    ).subscribe(storedState => {
      if (storedState && Object.keys(storedState).length > 0) {
        console.log("Successfully loaded state from backend:", storedState);
        const mergedState: InstallerState = { ...this.initialState };
        for (const key in storedState) {
          if (Object.prototype.hasOwnProperty.call(storedState, key) && key in mergedState) {
             if (key === 'appDeployImageConfig' && storedState.appDeployImageConfig && typeof storedState.appDeployImageConfig === 'object') {
              mergedState.appDeployImageConfig = { ...this.initialState.appDeployImageConfig, ...storedState.appDeployImageConfig };
            } else if (key === 'appDeployRegistryConfig' && storedState.appDeployRegistryConfig && typeof storedState.appDeployRegistryConfig === 'object') {
              mergedState.appDeployRegistryConfig = { ...this.initialState.appDeployRegistryConfig, ...storedState.appDeployRegistryConfig };
            } else if (key === 'appDeployGatewayConfig' && storedState.appDeployGatewayConfig && typeof storedState.appDeployGatewayConfig === 'object') {
              mergedState.appDeployGatewayConfig = { ...this.initialState.appDeployGatewayConfig, ...storedState.appDeployGatewayConfig };
            } else if (key === 'appDeployAdapterConfig' && storedState.appDeployAdapterConfig && typeof storedState.appDeployAdapterConfig === 'object') {
              mergedState.appDeployAdapterConfig = { ...this.initialState.appDeployAdapterConfig, ...storedState.appDeployAdapterConfig };
            } else {
              (mergedState as any)[key] = storedState[key];
            }
          }
        }
        this.installerStateSubject.next(mergedState);
       this.isStateLoadingSubject.next(false);
      } else {
        console.log("No state found in backend, initializing with default state.");
        this.installerStateSubject.next(this.initialState);
         this.resetState();
      }
       this.isStateLoadingSubject.next(false);
    });
  }


  private saveStateToBackend(): void {
    const currentState = this.installerStateSubject.getValue();
    this.apiService.storeBulkState(currentState).subscribe({
      next: () => console.log('State saved to backend successfully.'),
      error: (err) => console.error('Failed to save state to backend:', err)
    });
  }

  updateState(newState: Partial<InstallerState>): void {
    const currentState = this.installerStateSubject.getValue();
    const updatedState = { ...currentState, ...newState };
    this.installerStateSubject.next(updatedState);
    this.saveStateToBackend();
    console.log('Installer State Updated:', this.installerStateSubject.getValue());
  }

  getCurrentState(): InstallerState {
    return this.installerStateSubject.getValue();
  }

  resetState(): void {
    this.installerStateSubject.next(this.initialState);
    this.saveStateToBackend();
    console.log('Installer State Reset.');
  }

  updateCurrentStep(index: number): void {
    const currentState = this.getCurrentState();
    const highestStepReached = Math.max(currentState.highestStepReached, index);
    this.updateState({ currentStepIndex: index, highestStepReached });

  }

  updateInstallerGoal(goal: InstallerState['installerGoal']): void {
    this.updateState({ installerGoal: goal });
  }

  updateDeploymentGoal(goal: DeploymentGoal): void {
    this.updateState({ deploymentGoal: goal });
  }

  updatePrerequisitesMet(met: boolean): void {
    this.updateState({ prerequisitesMet: met });
  }

  updateGcpConfiguration(config: GcpConfiguration): void {
    this.updateState({ gcpConfiguration: config });
  }

  updateAppNameAndSize(appName: string, deploymentSize: DeploymentSize): void {
    this.updateState({ appName: appName, deploymentSize: deploymentSize });
  }

  setInfraDetails(details: InfraDetails): void {
    const externalIp = details.external_ip?.value || null;
    this.updateState({ infraDetails: details, appExternalIp: externalIp });
  }

  updateGlobalDomainConfig(config: DomainConfig): void {
    this.updateState({ globalDomainConfig: config });
  }

  updateComponentSubdomainPrefixes(prefixes: ComponentSubdomainPrefix[]): void {
    this.updateState({ componentSubdomainPrefixes: prefixes });
  }

  updateSubdomainConfigs(configs: SubdomainConfig[]): void {
    this.updateState({ subdomainConfigs: configs });
  }

  updateDockerImageConfigs(configs: DockerImageConfig[]): void {
    this.updateState({ dockerImageConfigs: configs });
  }

  updateAppSpecificConfigs(configs: AppSpecificConfig[]): void {
    this.updateState({ appSpecificConfigs: configs });
  }

  setAppExternalIp(ip: string | null): void {
    this.updateState({ appExternalIp: ip });
  }

  updateHealthCheckStatuses(statuses: HealthCheckStatus[]): void {
    this.updateState({ healthCheckStatuses: statuses });
  }

  updateHealthCheckStatus(serviceName: string, status: 'pending' | 'checking' | 'success' | 'failed', message?: string): void {
    const currentStatuses = this.getCurrentState().healthCheckStatuses;
    const existingIndex = currentStatuses.findIndex(s => s.serviceName === serviceName);

    if (existingIndex > -1) {
      const updatedStatuses = [...currentStatuses];
      updatedStatuses[existingIndex] = { ...updatedStatuses[existingIndex], status, message };
      this.updateState({ healthCheckStatuses: updatedStatuses });
    } else {
      const newStatus = { serviceName, status, message };
      this.updateState({ healthCheckStatuses: [...currentStatuses, newStatus] });
    }
  }

  updateDeploymentStatus(status: DeploymentStatus): void {
    this.updateState({ deploymentStatus: status });
  }

  updateDeployedServiceUrls(urls: { [key: string]: string }): void {
    this.updateState({ deployedServiceUrls: urls });
  }

  addDeploymentLog(log: string): void {
    const currentLogs = this.getCurrentState().deploymentLogs;
    this.updateState({ deploymentLogs: [...currentLogs, log] });
  }

  clearDeploymentLogs(): void {
    this.updateState({ deploymentLogs: [] });
  }

  updateAppDeployImageConfig(config: AppDeployImageConfig): void {
    const currentConfig = this.getCurrentState().appDeployImageConfig || this.initialState.appDeployImageConfig;
    this.updateState({ appDeployImageConfig: { ...currentConfig, ...config } });
  }

  updateAppDeployRegistryConfig(config: AppDeployRegistryConfig): void {
    const currentConfig = this.getCurrentState().appDeployRegistryConfig || this.initialState.appDeployRegistryConfig;
    this.updateState({ appDeployRegistryConfig: { ...currentConfig, ...config } });
  }

  updateAppDeployGatewayConfig(config: AppDeployGatewayConfig): void {
    const currentConfig = this.getCurrentState().appDeployGatewayConfig || this.initialState.appDeployGatewayConfig;
    this.updateState({ appDeployGatewayConfig: { ...currentConfig, ...config } });
  }

  updateAppDeployAdapterConfig(config: AppDeployAdapterConfig): void {
    const currentConfig = this.getCurrentState().appDeployAdapterConfig || this.initialState.appDeployAdapterConfig;
    this.updateState({ appDeployAdapterConfig: { ...currentConfig, ...config } });
  }

   setDeploymentState(isDeploying: boolean): void {
    this.isDeployingSubject.next(isDeploying);
  }

  updateAppDeploymentStatus(status: DeploymentStatus): void {
  this.updateState({ appDeploymentStatus: status });
}

}