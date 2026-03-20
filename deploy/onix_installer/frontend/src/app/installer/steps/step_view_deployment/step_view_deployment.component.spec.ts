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

import {beforeEach, bootstrap, describe, expect, it, setupModule,} from 'google3/javascript/angular2/testing/catalyst/fake_async';
import {of} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {InstallerState} from '../../types/installer.types';

import {StepViewDeployment} from './step_view_deployment.component';

describe('StepViewDeployment', () => {
  let installerStateServiceSpy: jasmine.SpyObj<InstallerStateService>;
  let webSocketServiceSpy: jasmine.SpyObj<WebSocketService>;

  beforeEach(() => {
    installerStateServiceSpy = jasmine.createSpyObj(
        'InstallerStateService',
        ['getCurrentState', 'updateState', 'updateAppDeploymentStatus'],
    );
    webSocketServiceSpy = jasmine.createSpyObj('WebSocketService', [
      'connect',
      'sendMessage',
      'closeConnection',
    ]);

    installerStateServiceSpy.getCurrentState.and.returnValue({
      isConfigChanged: false,
      isConfigLocked: false,
      isAppConfigValid: false,
      currentStepIndex: 9,
      highestStepReached: 9,
      installerGoal: 'create_new_open_network',
      deploymentGoal: {
        bap: false,
        bpp: false,
        gateway: false,
        registry: false,
        all: false,
      },
      prerequisitesMet: true,
      gcpConfiguration: null,
      appName: 'test-app',
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
      deploymentStatus: 'completed',
      appDeploymentStatus: 'pending',
      deploymentLogs: [],
      appDeployImageConfig: {
        registryImageUrl: '',
        registryAdminImageUrl: '',
        gatewayImageUrl: '',
        adapterImageUrl: '',
        subscriptionImageUrl: '',
      },
      appDeployRegistryConfig: {
        registryUrl: '',
        registrySubscriberId: '',
        registryKeyId: '',
        enableAutoApprover: false,
      },
      appDeployAdapterConfig: {
        enableSchemaValidation: false,
      },
      appDeployGatewayConfig: {
        gatewaySubscriptionId: '',
      },
      appDeploySecurityConfig: {
        enableInBoundAuth: false,
        issuerUrl: '',
        jwksFileContent: '',
        enableOutBoundAuth: false,
        audOverrides: '',
        idclaim: '',
        allowedValues: '',
      },
      lastDeployedAppPayload: null,
    } as unknown as InstallerState);
    webSocketServiceSpy.connect.and.returnValue(of({}));

    setupModule({
      imports: [StepViewDeployment],
      providers: [
        {provide: InstallerStateService, useValue: installerStateServiceSpy},
        {provide: WebSocketService, useValue: webSocketServiceSpy},
      ],
    });
  });

  it('should create', () => {
    const component = bootstrap(StepViewDeployment);
    expect(component).toBeTruthy();
  });
});
