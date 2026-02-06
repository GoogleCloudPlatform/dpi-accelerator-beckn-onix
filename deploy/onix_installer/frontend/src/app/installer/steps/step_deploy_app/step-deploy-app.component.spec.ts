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
import {MatTabsModule} from '@angular/material/tabs';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {BehaviorSubject, EMPTY, of, Subject, throwError} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {DeploymentGoal, DeploymentStatus, InstallerState} from '../../types/installer.types';

import {StepAppDeployComponent} from './step-deploy-app.component';

const initialMockState: InstallerState = {
    currentStepIndex: 6,
    highestStepReached: 6,
    installerGoal: 'create_new_open_network',
    deploymentGoal: { all: true, gateway: true, registry: true, bap: true, bpp: true },
    prerequisitesMet: true,
    gcpConfiguration: { projectId: 'test-project', region: 'us-central1' },
    appName: 'onix-app',
    deploymentSize: 'small',
    infraDetails: {
        external_ip: { value: '1.2.3.4' },
        registry_url: { value: 'https://infra-registry.com' }
    },
    appExternalIp: '1.2.3.4',
    globalDomainConfig: {
        domainType: 'other_domain',
        baseDomain: 'example.com',
        dnsZone: 'example-zone'
    },
    subdomainConfigs: [
        { component: 'registry', subdomainName: 'registry.example.com', domainType: 'google_domain' },
        { component: 'gateway', subdomainName: 'gateway.example.com', domainType: 'google_domain' },
        { component: 'adapter', subdomainName: 'adapter.example.com', domainType: 'google_domain' },
        { component: 'subscriber', subdomainName: 'sub.example.com', domainType: 'google_domain' }
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
    appDeployGatewayConfig: {
        gatewaySubscriptionId: 'gw-sub-id'
    },
    appDeployAdapterConfig: {
        enableSchemaValidation: true
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


describe('StepAppDeployComponent', () => {
    let component: StepAppDeployComponent;
    let fixture: ComponentFixture<StepAppDeployComponent>;
    let installerStateService: MockInstallerStateService;
    let webSocketService: MockWebSocketService;
    let router: Router;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                StepAppDeployComponent,
                NoopAnimationsModule,
                ReactiveFormsModule,
                MatTabsModule
            ],
            providers: [
                { provide: InstallerStateService, useClass: MockInstallerStateService },
                { provide: WebSocketService, useClass: MockWebSocketService },
                { provide: Router, useValue: { navigate: jasmine.createSpy('navigate') } },
                { provide: Clipboard, useClass: MockClipboard },
                FormBuilder
            ]
        }).compileComponents();

        fixture = TestBed.createComponent(StepAppDeployComponent);
        component = fixture.componentInstance;
        installerStateService = TestBed.inject(InstallerStateService) as any;
        webSocketService = TestBed.inject(WebSocketService) as any;
        router = TestBed.inject(Router);
    });

    it('should create', () => {
        fixture.detectChanges();
        expect(component).toBeTruthy();
    });

    it('should load success data from state on initialization if app deployment is already complete', () => {
        const successData = {
            deployedServiceUrls: { gateway: 'https://gateway.example.com' },
            servicesDeployed: ['gateway', 'registry'],
            logsExplorerUrls: { gateway: 'https://logs.gcp.com/gateway' },
            appExternalIp: '1.2.3.4'
        };

        // Set the state to 'completed' before the component initializes
        installerStateService.setState({
            appDeploymentStatus: 'completed',
            ...successData
        });

        fixture.detectChanges();

        const configElement = fixture.nativeElement.querySelector('.app-deploy-config');
        const statusElement = fixture.nativeElement.querySelector('.app-deploy-status-section');

        expect(configElement).toBeFalsy();
        expect(statusElement).toBeTruthy();

        expect(component.serviceUrls).toEqual(successData.deployedServiceUrls);
        expect(component.servicesDeployed).toEqual(successData.servicesDeployed);
    });

    it('should start deployment and update state to in-progress', () => {
        fixture.detectChanges();
        spyOn(component.deploymentInitiated, 'emit');
        component.onDeployApp();

        expect(installerStateService.updateAppDeploymentStatus).toHaveBeenCalledWith('in-progress');
        expect(component.isAppDeploying).toBeTrue(); // Check the getter
        expect(component.deploymentInitiated.emit).toHaveBeenCalled();
        expect(webSocketService.connect).toHaveBeenCalledWith('ws://localhost:8000/ws/deployApp');
    });

    it('should handle success messages and update state to completed', () => {
        fixture.detectChanges();
        spyOn(component.deploymentComplete, 'emit');

        const successData = {
            service_urls: { gateway: 'https://gateway.example.com' },
            services_deployed: ['gateway', 'registry'],
            logs_explorer_urls: { gateway: 'https://logs.gcp.com/gateway' },
            app_external_ip: '1.2.3.4'
        };

        component.onDeployApp();
        webSocketService.sendWsMessage({ type: 'success', data: successData });
        fixture.detectChanges();

        expect(installerStateService.updateAppDeploymentStatus).toHaveBeenCalledWith('completed');
        expect(installerStateService.updateState).toHaveBeenCalledWith({
            deployedServiceUrls: successData.service_urls,
            servicesDeployed: successData.services_deployed,
            logsExplorerUrls: successData.logs_explorer_urls,
            appExternalIp: successData.app_external_ip
        });
        expect(component.appDeploymentComplete).toBeTrue();
        expect(component.deploymentComplete.emit).toHaveBeenCalled();
    });

    it('should handle error messages and update state to failed', () => {
        fixture.detectChanges();
        spyOn(component.deploymentError, 'emit');
        component.onDeployApp();
        webSocketService.sendWsMessage({ type: 'error', message: 'Deployment failed badly' });
        fixture.detectChanges();

        expect(installerStateService.updateAppDeploymentStatus).toHaveBeenCalledWith('failed');
        expect(component.appDeploymentFailed).toBeTrue();
        expect(component.deploymentError.emit).toHaveBeenCalledWith('Deployment failed badly');
    });

    it('should handle WebSocket connection errors and update state to failed', () => {
        fixture.detectChanges();
        spyOn(component.deploymentError, 'emit');
        webSocketService.connect.and.returnValue(throwError(() => new Error('Connection failed')));

        component.onDeployApp();
        fixture.detectChanges();

        expect(installerStateService.updateAppDeploymentStatus).toHaveBeenCalledWith('failed');
        expect(component.appDeploymentFailed).toBeTrue();
        expect(component.deploymentError.emit).toHaveBeenCalledWith(jasmine.stringContaining('WebSocket connection error'));
    });

    it('should reset deployment state on resetDeployment()', () => {
        installerStateService.setState({ appDeploymentStatus: 'completed' });
        fixture.detectChanges();

        expect(component.appDeploymentComplete).toBeTrue();

        component.resetDeployment();
        fixture.detectChanges();

        expect(installerStateService.updateAppDeploymentStatus).toHaveBeenCalledWith('pending');
        expect(component.appDeploymentComplete).toBeFalse();
    });
});