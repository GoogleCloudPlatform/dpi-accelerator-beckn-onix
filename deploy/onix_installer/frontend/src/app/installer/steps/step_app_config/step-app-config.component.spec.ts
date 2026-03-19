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

import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {MatSlideToggleModule} from '@angular/material/slide-toggle';
import {MatTabsModule} from '@angular/material/tabs';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {BehaviorSubject, EMPTY, of, Subject, throwError} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {AppDeploySecurityConfig, DeploymentGoal, DeploymentStatus, InstallerState} from '../../types/installer.types';

import {StepAppConfigComponent} from './step-app-config.component';

const initialMockState: InstallerState = {
  isConfigChanged: false,
  isConfigLocked: false,
  isAppConfigValid: false,
  currentStepIndex: 6,
  highestStepReached: 6,
  installerGoal: 'create_new_open_network',
  deploymentGoal:
      {all: true, gateway: true, registry: true, bap: true, bpp: true},
  prerequisitesMet: true,
  gcpConfiguration: {projectId: 'test-project', region: 'us-central1'},
  appName: 'onix-app',
  deploymentSize: 'small',
  infraDetails: {
    external_ip: {value: '1.2.3.4'},
    registry_url: {value: 'https://infra-registry.com'}
  },
  appExternalIp: '1.2.3.4',
  globalDomainConfig: {
    domainType: 'other_domain',
    baseDomain: 'example.com',
    dnsZone: 'example-zone'
  },
  subdomainConfigs: [
    {
      component: 'registry',
      subdomainName: 'registry.example.com',
      domainType: 'google_domain'
    },
    {
      component: 'gateway',
      subdomainName: 'gateway.example.com',
      domainType: 'google_domain'
    },
    {
      component: 'adapter',
      subdomainName: 'adapter.example.com',
      domainType: 'google_domain'
    },
    {
      component: 'subscriber',
      subdomainName: 'sub.example.com',
      domainType: 'google_domain'
    }
  ],
  appDeployImageConfig: {
    registryImageUrl: 'reg-img:v1',
    registryAdminImageUrl: 'reg-admin-img:v1',
    gatewayImageUrl: 'gw-img:v1',
    adapterImageUrl: 'adapter-img:v1',
    subscriptionImageUrl: 'sub-img:v1'
  },
  appDeployRegistryConfig: {
    registryUrl: 'https://my-registry.com',
    registryKeyId: 'my-key-id',
    registrySubscriberId: 'my-sub-id',
    enableAutoApprover: true
  },
  appDeployGatewayConfig: {gatewaySubscriptionId: 'gw-sub-id'},
  appDeployAdapterConfig: {enableSchemaValidation: true},
  appDeploySecurityConfig: {
    enableInBoundAuth: true,
    enableOutBoundAuth: true,
    issuerUrl: 'https://issuer.com',
    idclaim: 'sub',
    allowedValues: 'val1',
    jwksFileContent: '',
    audOverrides: 'aud1,aud2'
  },
  healthCheckStatuses: [],
  deploymentStatus: 'completed',
  appDeploymentStatus: 'pending',
  deploymentLogs: [],
  deployedServiceUrls: {},
  servicesDeployed: [],
  logsExplorerUrls: {},
  dockerImageConfigs: [],
  appSpecificConfigs: [],
  componentSubdomainPrefixes: [],
  lastDeployedAppPayload: null as any
};

class MockInstallerStateService {
  private state = new BehaviorSubject<InstallerState>(
      JSON.parse(JSON.stringify(initialMockState)) as InstallerState);
  installerState$ = this.state.asObservable();

  getCurrentState = () => this.state.getValue();
  updateAppDeploymentStatus =
      jasmine.createSpy('updateAppDeploymentStatus')
          .and.callFake((status: DeploymentStatus) => {
            this.setState({appDeploymentStatus: status});
          });
  updateState = jasmine.createSpy('updateState')
                    .and.callFake((newState: Partial<InstallerState>) => {
                      this.setState(newState);
                    });

  updateAppDeployImageConfig = jasmine.createSpy('updateAppDeployImageConfig');
  updateAppDeployRegistryConfig =
      jasmine.createSpy('updateAppDeployRegistryConfig');
  updateAppDeployGatewayConfig =
      jasmine.createSpy('updateAppDeployGatewayConfig');
  updateAppDeployAdapterConfig =
      jasmine.createSpy('updateAppDeployAdapterConfig');
  updateAppDeploySecurityConfig =
      jasmine.createSpy('updateAppDeploySecurityConfig');

  setState(newState: Partial<InstallerState>) {
    const currentState = this.state.getValue();
    this.state.next({...currentState, ...newState});
  }
}

class MockWebSocketService {
    private messageSubject = new Subject<any>();
    connect = jasmine.createSpy('connect').and.returnValue(this.messageSubject.asObservable());
    sendMessage = jasmine.createSpy('sendMessage');
    closeConnection = jasmine.createSpy('closeConnection');

    sendWsMessage(message: any) { this.messageSubject.next(message); }
    simulateWsError(error: any) { this.messageSubject.error(error); }
}

class MockClipboard {
    copy = jasmine.createSpy('copy');
}


describe('StepAppConfigComponent', () => {
  let component: StepAppConfigComponent;
  let fixture: ComponentFixture<StepAppConfigComponent>;
  let installerStateService: MockInstallerStateService;
  let webSocketService: MockWebSocketService;
  let router: Router;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [
            StepAppConfigComponent, NoopAnimationsModule, ReactiveFormsModule,
            MatTabsModule, MatSlideToggleModule
          ],
          providers: [
            {
              provide: InstallerStateService,
              useClass: MockInstallerStateService
            },
            {provide: WebSocketService, useClass: MockWebSocketService}, {
              provide: Router,
              useValue: {navigate: jasmine.createSpy('navigate')}
            },
            {provide: Clipboard, useClass: MockClipboard}, FormBuilder
          ]
        })
        .compileComponents();

    fixture = TestBed.createComponent(StepAppConfigComponent);
    component = fixture.componentInstance;
    installerStateService = TestBed.inject(InstallerStateService) as any;
    webSocketService = TestBed.inject(WebSocketService) as any;
    router = TestBed.inject(Router);
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });
});