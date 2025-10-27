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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { BehaviorSubject, Subject, throwError } from 'rxjs';

import { StepHealthCheck } from './step-health-check';
import { InstallerStateService } from '../../../core/services/installer-state.service';
import { WebSocketService } from '../../../core/services/websocket.service';
import { InstallerState } from '../../types/installer.types';


const initialMockState: InstallerState = {
  currentStepIndex: 8,
  installerGoal: 'create_new_open_network',
  prerequisitesMet: true,
  deploymentGoal: { all: true, gateway: true, registry: true, bap: true, bpp: true },
  gcpConfiguration: { projectId: 'test-project', region: 'us-central1' },
  appName: 'onix-app',
  deploymentSize: 'small',
  deploymentStatus: 'completed',
  deploymentLogs: [],
  infraDetails: null,
  appExternalIp: '1.2.3.4',
  componentSubdomainPrefixes: [
    { component: 'registry', subdomainPrefix: 'registry.example.com' },
    { component: 'registry_admin', subdomainPrefix: 'admin.example.com' },
    { component: 'gateway', subdomainPrefix: 'gateway.example.com' },
    { component: 'adapter', subdomainPrefix: 'adapter.example.com' },
    { component: 'subscriber', subdomainPrefix: 'subscriber.example.com' }
  ],
  subdomainConfigs: [],
  appSpecificConfigs: [],
  dockerImageConfigs: [],
  healthCheckStatuses: [],
  deployedServiceUrls: {},
  appDeployImageConfig: {
    registryImageUrl: '', registryAdminImageUrl: '', gatewayImageUrl: '', adapterImageUrl: '', subscriptionImageUrl: ''
  },
  appDeployRegistryConfig: {
    registryUrl: '', registryKeyId: '', registrySubscriberId: '', enableAutoApprover: false
  },
  appDeployGatewayConfig: { gatewaySubscriptionId: '' },
  appDeployAdapterConfig: { enableSchemaValidation: false },
  globalDomainConfig: null,
  highestStepReached: 8,
  appDeploymentStatus: 'completed',
  servicesDeployed: [],
  logsExplorerUrls: {}
};

class MockInstallerStateService {
  private state = new BehaviorSubject<InstallerState>(initialMockState);
  installerState$ = this.state.asObservable();
  getCurrentState = () => this.state.getValue();
}

class MockWebSocketService {
  connectionSubject = new Subject<any>();
  connect = jasmine.createSpy('connect').and.returnValue(this.connectionSubject.asObservable());
  sendMessage = jasmine.createSpy('sendMessage');
  closeConnection = jasmine.createSpy('closeConnection');

  receiveMessage(message: any) {
    this.connectionSubject.next(message);
  }

  simulateError(error: any) {
    this.connectionSubject.error(error);
  }
}

describe('StepHealthCheck', () => {
  let component: StepHealthCheck;
  let fixture: ComponentFixture<StepHealthCheck>;
  let installerStateService: MockInstallerStateService;
  let webSocketService: MockWebSocketService;
  let router: Router;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StepHealthCheck, NoopAnimationsModule],
      providers: [
        { provide: InstallerStateService, useClass: MockInstallerStateService },
        { provide: WebSocketService, useClass: MockWebSocketService },
        { provide: Router, useValue: { navigate: jasmine.createSpy('navigate') } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StepHealthCheck);
    component = fixture.componentInstance;
    installerStateService = TestBed.inject(InstallerStateService) as any;
    webSocketService = TestBed.inject(WebSocketService) as any;
    router = TestBed.inject(Router);
    fixture.detectChanges();
  });

  it('should create and initialize checks from state', () => {
    expect(component).toBeTruthy();
    // With 'all: true', we expect all 5 services to be configured
    expect(component.checkResults.length).toBe(5);
    expect(component.checkResults.some(c => c.name === 'Gateway')).toBeTrue();
    expect(component.checkResults.find(c => c.name === 'Registry')?.url).toBe('registry.example.com');
  });

  it('should start health check process on runHealthCheck()', () => {
    component.runHealthCheck();
    fixture.detectChanges();

    expect(component.healthCheckStatus).toBe('inProgress');
    expect(webSocketService.connect).toHaveBeenCalledWith('ws://localhost:8000/ws/healthCheck');
    expect(webSocketService.sendMessage).toHaveBeenCalled();
    expect(component.checkResults.every(c => c.status === 'pending')).toBeTrue();
  });

  it('should handle individual service success message from WebSocket', () => {
    component.runHealthCheck();
    const successMessage = { service: 'Gateway', type: 'success', message: 'Gateway is healthy' };
    webSocketService.receiveMessage(successMessage);
    fixture.detectChanges();

    const gatewayCheck = component.checkResults.find(c => c.name === 'Gateway');
    expect(gatewayCheck?.status).toBe('success');
    expect(component.logMessages).toContain('Gateway is healthy');
  });

  it('should handle individual service error message from WebSocket', () => {
    component.runHealthCheck();
    const errorMessage = { service: 'Registry', type: 'error', message: 'Registry connection failed' };
    webSocketService.receiveMessage(errorMessage);
    fixture.detectChanges();

    const registryCheck = component.checkResults.find(c => c.name === 'Registry');
    expect(registryCheck?.status).toBe('failed');
  });

  it('should handle final success message and update all statuses', () => {
    component.runHealthCheck();
    webSocketService.receiveMessage({ service: 'Gateway', type: 'success', message: 'Gateway OK' });
    fixture.detectChanges();
    const finalSuccessMessage = { action: 'all_services_healthy', message: 'All services are responsive.' };
    webSocketService.receiveMessage(finalSuccessMessage);
    fixture.detectChanges();

    expect(component.healthCheckStatus).toBe('success');
    expect(component.showSuccessModal).toBeTrue();
    expect(component.checkResults.every(c => c.status === 'success')).toBeTrue();
    expect(webSocketService.closeConnection).toHaveBeenCalled();
  });

  it('should handle timeout message and mark pending as failed', () => {
    component.runHealthCheck();
    webSocketService.receiveMessage({ service: 'Gateway', type: 'success', message: 'Gateway OK' });
    fixture.detectChanges();

    const timeoutMessage = { action: 'health_check_timeout', message: 'Health check timed out.' };
    webSocketService.receiveMessage(timeoutMessage);
    fixture.detectChanges();

    expect(component.healthCheckStatus).toBe('failed');
    const gatewayCheck = component.checkResults.find(c => c.name === 'Gateway');
    const registryCheck = component.checkResults.find(c => c.name === 'Registry');

    expect(gatewayCheck?.status).toBe('success');
    expect(registryCheck?.status).toBe('failed');
    expect(webSocketService.closeConnection).toHaveBeenCalled();
  });

  it('should handle WebSocket connection error', () => {
    webSocketService.connect.and.returnValue(throwError(() => new Error('Connection failed')));
    component.runHealthCheck();
    fixture.detectChanges();

    expect(component.healthCheckStatus).toBe('failed');
  });

  it('should navigate to the next step on onNext()', () => {
    component.onNext();
    expect(router.navigate).toHaveBeenCalledWith(['installer', 'subscribe']);
  });
});