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

import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {BehaviorSubject} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {ComponentSubdomainPrefix, DomainConfig, InstallerState} from '../../types/installer.types';

import {StepDomainConfigComponent} from './step-domain-configuration.component';

const initialMockState: InstallerState = {
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
  highestStepReached: 5,
  appDeploymentStatus: 'pending',
  servicesDeployed: [],
  logsExplorerUrls: {}
};

// Mock Services
class MockInstallerStateService {
  private state = new BehaviorSubject<InstallerState>(initialMockState);
  installerState$ = this.state.asObservable();

  updateComponentSubdomainPrefixes =
      jasmine.createSpy('updateComponentSubdomainPrefixes');
  updateGlobalDomainConfig = jasmine.createSpy('updateGlobalDomainConfig');
  updateSubdomainConfigs = jasmine.createSpy('updateSubdomainConfigs');

  // Helper to update the mock state for testing
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
    fixture.detectChanges();  // Trigger ngOnInit
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize component prefixes form array based on deploymentGoal',
     () => {
       // Default 'all: true' should create all 5 prefixes
       expect(component.componentPrefixes.length).toBe(5);
       expect(component.componentsToConfigure).toContain('registry');
       expect(component.componentsToConfigure).toContain('gateway');
       expect(component.componentsToConfigure).toContain('adapter');
     });

  it('should correctly toggle validators for "google_domain"', () => {
    const globalDetails = component.globalDomainDetailsFormGroup;
    globalDetails.get('domainType')?.setValue('google_domain');
    fixture.detectChanges();

    expect(globalDetails.get('baseDomain')?.hasValidator(Validators.required))
        .toBeTrue();
    expect(globalDetails.get('dnsZone')?.hasValidator(Validators.required))
        .toBeTrue();
    expect(globalDetails.get('actionRequiredAcknowledged')?.validator)
        .toBeNull();
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

  describe('Internal Navigation: onNextInternal', () => {
    it('should not advance from step 1 if prefixes are invalid', () => {
      component.currentInternalStep = 1;
      component.componentPrefixes.at(0).get('subdomainPrefix')?.setValue('');
      fixture.detectChanges();

      component.onNextInternal();
      expect(component.currentInternalStep).toBe(1);
      expect(installerStateService.updateComponentSubdomainPrefixes)
          .not.toHaveBeenCalled();
    });
  });
});