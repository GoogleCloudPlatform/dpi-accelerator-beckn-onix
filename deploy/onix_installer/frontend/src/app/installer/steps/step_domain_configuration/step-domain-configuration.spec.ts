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

import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {FormBuilder, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {BehaviorSubject} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {ComponentSubdomainPrefix, DomainConfig, InstallerState} from '../../types/installer.types';

import {StepDomainConfigComponent} from './step-domain-configuration.component';

// Prevent js_scrub from stripping the imports
const _dummyStepDomainConfigComponent = StepDomainConfigComponent;

const initialMockState: InstallerState = {
  isConfigChanged: false,
  isConfigLocked: false,
  isAppConfigValid: false,
  currentStepIndex: 5,
  installerGoal: 'create_new_open_network',
  prerequisitesMet: true,
  deploymentGoal:
      {all: true, gateway: false, registry: false, bap: false, bpp: false},
  gcpConfiguration: {projectId: 'test-project', region: 'us-central1'},
  appName: 'onix-app',
  deploymentSize: 'small',
  deploymentStatus: 'completed',
  deploymentLogs: [],
  infraDetails: null,
  appExternalIp: null,
  globalDomainConfig: null,
  componentSubdomainPrefixes: [],
  subdomainConfigs: [],
  dockerImageConfigs: [],
  appSpecificConfigs: [],
  healthCheckStatuses: [],
  deployedServiceUrls: {},
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
  appDeployGatewayConfig: {gatewaySubscriptionId: ''},
  appDeployAdapterConfig: {enableSchemaValidation: false},
  appDeploySecurityConfig: {
    enableInBoundAuth: false,
    enableOutBoundAuth: false,
    issuerUrl: '',
    idclaim: '',
    allowedValues: '',
    jwksFileContent: '',
    audOverrides: ''
  },
  highestStepReached: 5,
  appDeploymentStatus: 'pending',
  servicesDeployed: [],
  logsExplorerUrls: {},
  lastDeployedAppPayload: null as any,
  enableCloudArmor: false,
  cloudArmorRateLimit: 100
};

class MockInstallerStateService {
  private state = new BehaviorSubject<InstallerState>(initialMockState);
  installerState$ = this.state.asObservable();

  updateComponentSubdomainPrefixes =
      jasmine.createSpy('updateComponentSubdomainPrefixes');
  updateGlobalDomainConfig = jasmine.createSpy('updateGlobalDomainConfig');
  updateSubdomainConfigs = jasmine.createSpy('updateSubdomainConfigs');

  getCurrentState() {
    return this.state.getValue();
  }

  setState(newState: Partial<InstallerState>) {
    const currentState = this.state.getValue();
    this.state.next({...currentState, ...newState});
  }
}

describe('StepDomainConfigComponent', () => {
  let component: StepDomainConfigComponent;
  let fixture: ComponentFixture<StepDomainConfigComponent>;
  let installerStateService: MockInstallerStateService;
  let router: Router;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [
            StepDomainConfigComponent, ReactiveFormsModule, NoopAnimationsModule
          ],
          providers: [
            FormBuilder, {
              provide: InstallerStateService,
              useClass: MockInstallerStateService
            },
            {
              provide: Router,
              useValue: {navigate: jasmine.createSpy('navigate')}
            }
          ]
        })
        .compileComponents();

    fixture = TestBed.createComponent(StepDomainConfigComponent);
    component = fixture.componentInstance;
    installerStateService = TestBed.inject(InstallerStateService) as any;
    router = TestBed.inject(Router);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize component prefixes form array based on deploymentGoal',
     () => {
       expect(component.componentPrefixes.length).toBe(5);
     });

  it('should correctly toggle validators for "google_domain"', () => {
    const globalDetails = component.globalDomainDetailsFormGroup;
    globalDetails.get('domainType')?.setValue('google_domain');
    fixture.detectChanges();

    expect(globalDetails.get('baseDomain')?.hasValidator(Validators.required))
        .toBeTrue();
    expect(globalDetails.get('dnsZone')?.hasValidator(Validators.required))
        .toBeTrue();
  });

  it('should correctly toggle validators for "other_domain"', () => {
    const globalDetails = component.globalDomainDetailsFormGroup;
    globalDetails.get('domainType')?.setValue('other_domain');
    fixture.detectChanges();

    expect(globalDetails.get('dnsZone')?.validator).toBeNull();
    expect(globalDetails.get('actionRequiredAcknowledged')
               ?.hasValidator(Validators.requiredTrue))
        .toBeTrue();
  });

  it('should handle onTabChange', () => {
    component.currentInternalStep = 1;

    // Simulate tab change event
    const event = {index: 1} as any;  // Change to step 2
    component.onTabChange(event);

    // Since isStep2Enabled depends on componentPrefixes being valid.
    // Default prefixes are valid (have values), so it should allow step 2!
    expect(component.currentInternalStep).toBe(2);
  });

  it('should handle onNextInternal for step 1 (valid)', fakeAsync(() => {
       component.currentInternalStep = 1;
       fixture.detectChanges();

       component.onNextInternal();
       tick();  // Process setTimeout(0)

       expect(installerStateService.updateComponentSubdomainPrefixes)
           .toHaveBeenCalledWith(component.componentPrefixes.value);
       expect(component.currentInternalStep)
           .toBe(2);  // Should advance to step 2
     }));

  it('should handle onBackInternal', () => {
    component.currentInternalStep = 2;
    component.onBackInternal();
    expect(component.currentInternalStep).toBe(1);

    component.onBackInternal();  // Now it's 1, should navigate back
    expect(router.navigate).toHaveBeenCalledWith(['installer', 'deploy-infra']);
  });

  it('getErrorMessage should return correct message', () => {
    const control =
        component.domainConfigForm.get('globalDomainDetails.baseDomain');
    control?.markAsTouched();
    control?.setErrors({required: true});

    expect(component.getErrorMessage(control, 'Base Domain'))
        .toBe('Base Domain is required.');

    control?.setErrors({requiredTrue: true});
    expect(component.getErrorMessage(control, 'Base Domain'))
        .toBe('Please acknowledge this action to proceed.');
  });
});