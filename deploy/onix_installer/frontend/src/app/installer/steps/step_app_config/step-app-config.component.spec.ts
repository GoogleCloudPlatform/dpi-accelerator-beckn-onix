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

import {Clipboard} from '@angular/cdk/clipboard';
import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {MatSlideToggleModule} from '@angular/material/slide-toggle';
import {MatTabsModule} from '@angular/material/tabs';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {BehaviorSubject, EMPTY, of, Subject, throwError} from 'rxjs';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {AppDeploySecurityConfig, DeploymentGoal, DeploymentStatus, InstallerState} from '../../types/installer.types';

import {StepAppConfigComponent} from './step-app-config.component';

// Prevent js_scrub from stripping the imports
const _dummyStepAppConfigComponent = StepAppConfigComponent;

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
  lastDeployedAppPayload: null as any,
  enableCloudArmor: false,
  cloudArmorRateLimit: 100
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
  let apiServiceSpy: jasmine.SpyObj<ApiService>;
  let router: Router;

  beforeEach(async () => {
    apiServiceSpy = jasmine.createSpyObj('ApiService', ['postConfigs']);
    apiServiceSpy.postConfigs.and.returnValue(of({}));

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
            {provide: ApiService, useValue: apiServiceSpy},
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

  it('should handle onJwkFileSelected with valid file', () => {
    fixture.detectChanges();
    const file =
        new File(['{"keys":[]}'], 'test-jwks.json', {type: 'application/json'});
    const event = {target: {files: [file]}};

    component.onJwkFileSelected(event);

    expect(component.selectedJwkFileName).toBe('test-jwks.json');
    expect(component.securityConfigForm.get('jwksFile')?.value).toBe(file);
    expect(component.securityConfigForm.get('jwksFile')?.touched).toBeTrue();
  });

  it('should handle onJwkFileSelected when no file is selected', () => {
    fixture.detectChanges();
    const event = {target: {files: []}};

    component.onJwkFileSelected(event);

    expect(component.selectedJwkFileName).toBeUndefined();
    expect(component.securityConfigForm.get('jwksFile')?.value).toBe('');
  });

  it('should handle clearJwkFile', () => {
    fixture.detectChanges();

    // Setup initial state
    component.selectedJwkFileName = 'test-jwks.json';
    component.securityConfigForm.patchValue({
      jwksFile: new File(
          ['{"keys":[]}'], 'test-jwks.json', {type: 'application/json'})
    });
    component.jwkFileInput = {nativeElement: {value: 'test-jwks.json'}} as any;

    const event = new Event('click');
    spyOn(event, 'stopPropagation');

    component.clearJwkFile(event);

    expect(event.stopPropagation).toHaveBeenCalled();
    expect(component.selectedJwkFileName).toBeUndefined();
    expect(component.securityConfigForm.get('jwksFile')?.value).toBe('');
    expect(component.jwkFileInput.nativeElement.value).toBe('');
    expect(installerStateService.updateAppDeploySecurityConfig)
        .toHaveBeenCalled();
  });

  it('should proceed to config generation and call apiService.postConfigs',
     fakeAsync(() => {
       fixture.detectChanges();

       // Simulate valid form
       spyOnProperty(component, 'isAppConfigValid', 'get')
           .and.returnValue(true);

       component.proceedToConfigGeneration();
       tick();

       expect(apiServiceSpy.postConfigs).toHaveBeenCalled();
       expect(installerStateService.updateState).toHaveBeenCalledWith({
         isAppConfigValid: true
       });
       expect(router.navigate).toHaveBeenCalledWith([
         'installer', 'view-config'
       ]);
       expect(component.isGeneratingConfigs).toBeFalse();
     }));

  it('should read JWKS file and proceed to config generation', fakeAsync(() => {
       fixture.detectChanges();

       const jwksJson = {keys: [{kty: 'RSA'}]};
       const file = new File(
           [JSON.stringify(jwksJson)], 'test-jwks.json',
           {type: 'application/json'});

       component.securityConfigForm.patchValue({
         enableInBoundAuth: true,
         jwksFile: file,
         issuerUrl: 'https://issuer.com',
         idclaim: 'sub',
         allowedValues: 'val1,val2'
       });

       spyOnProperty(component, 'isAppConfigValid', 'get')
           .and.returnValue(true);

       spyOn<any>(component, 'readFileContent')
           .and.returnValue(Promise.resolve(JSON.stringify(jwksJson)));

       component.proceedToConfigGeneration();
       tick();

       expect(apiServiceSpy.postConfigs).toHaveBeenCalled();
       const postArgs = apiServiceSpy.postConfigs.calls.mostRecent().args[0];
       expect(postArgs.security_config.jwks_content)
           .toContain('{\\"keys\\":[{\\"kty\\":\\"RSA\\"}]}');
       expect(postArgs.security_config.allowed_values).toEqual([
         'val1', 'val2'
       ]);
       expect(component.isGeneratingConfigs).toBeFalse();
     }));

  it('should not proceed to config generation if form is invalid',
     fakeAsync(() => {
       fixture.detectChanges();

       // Simulate invalid form
       spyOnProperty(component, 'isAppConfigValid', 'get')
           .and.returnValue(false);

       component.proceedToConfigGeneration();
       tick();

       expect(apiServiceSpy.postConfigs).not.toHaveBeenCalled();
     }));

  it('should handle config generation error', fakeAsync(() => {
       fixture.detectChanges();

       apiServiceSpy.postConfigs.and.returnValue(
           throwError(() => ({status: 500})));
       spyOnProperty(component, 'isAppConfigValid', 'get')
           .and.returnValue(true);

       component.proceedToConfigGeneration();
       tick();

       expect(apiServiceSpy.postConfigs).toHaveBeenCalled();
       expect(component.configGenerationError)
           .toContain('Failed to generate configurations');
       expect(component.isGeneratingConfigs).toBeFalse();
     }));

  it('should dynamically update security form validators based on enableInBoundAuth',
     () => {
       fixture.detectChanges();

       const issuerUrlCtrl = component.securityConfigForm.get('issuerUrl');
       const idClaimCtrl = component.securityConfigForm.get('idclaim');
       const allowedValuesCtrl =
           component.securityConfigForm.get('allowedValues');

       // Default or disabled state
       component.securityConfigForm.patchValue({enableInBoundAuth: false});
       expect(issuerUrlCtrl?.hasValidator(Validators.required)).toBeFalse();
       expect(idClaimCtrl?.hasValidator(Validators.required)).toBeFalse();
       expect(allowedValuesCtrl?.hasValidator(Validators.required)).toBeFalse();

       // Enabled state
       component.securityConfigForm.patchValue({enableInBoundAuth: true});
       expect(issuerUrlCtrl?.hasValidator(Validators.required)).toBeTrue();
       expect(idClaimCtrl?.hasValidator(Validators.required)).toBeTrue();
       expect(allowedValuesCtrl?.hasValidator(Validators.required)).toBeTrue();
     });

  it('should dynamically update security form validators based on enableOutBoundAuth',
     () => {
       fixture.detectChanges();

       const audOverridesCtrl =
           component.securityConfigForm.get('audOverrides');

       component.securityConfigForm.patchValue({enableOutBoundAuth: true});
       expect(audOverridesCtrl?.validator).toBeNull();
     });

  it('should compute isAppConfigValid correctly based on form states', () => {
    fixture.detectChanges();

    component.imageConfigForm.patchValue({
      registryImageUrl: 'img1',
      registryAdminImageUrl: 'img2',
      gatewayImageUrl: 'img3',
      adapterImageUrl: 'img4',
      subscriptionImageUrl: 'img5',
    });
    component.registryConfigForm.patchValue({
      registryUrl: 'http://test.com',
      registryKeyId: 'key',
      registrySubscriberId: 'sub',
      enableAutoApprover: false
    });
    component.gatewayConfigForm.patchValue({
      gatewaySubscriptionId: 'gw-sub',
    });
    component.adapterConfigForm.patchValue({
      enableSchemaValidation: false,
    });
    component.securityConfigForm.patchValue({
      enableInBoundAuth: false,
      enableOutBoundAuth: false,
      issuerUrl: '',
      idclaim: '',
      allowedValues: '',
      jwksFile: '',
      audOverrides: '',
    });

    expect(component.isAppConfigValid).toBeTrue();

    // Make an image field invalid
    component.imageConfigForm.patchValue({registryImageUrl: ''});
    expect(component.isAppConfigValid).toBeFalse();

    // Revert image form and make registry form invalid
    component.imageConfigForm.patchValue({registryImageUrl: 'img1'});
    component.registryConfigForm.patchValue({registryUrl: 'invalid-url'});
    expect(component.isAppConfigValid).toBeFalse();
  });

  it('should validate jwksFile as invalid if content is not valid JSON',
     fakeAsync(() => {
       fixture.detectChanges();
       const jwksFileCtrl = component.securityConfigForm.get('jwksFile');

       const invalidJsonFile = new File(
           ['{ invalid json }'], 'invalid.json', {type: 'application/json'});

       const mockFileReader = {
         readAsText:
             jasmine.createSpy('readAsText').and.callFake(function(this: any) {
               this.result = '{ invalid json }';
               if (this.onload) this.onload();
             })
       };
       spyOn(window as any, 'FileReader').and.returnValue(mockFileReader);

       jwksFileCtrl?.setValue(invalidJsonFile);

       tick();  // Wait for async validator to complete

       expect(jwksFileCtrl?.hasError('invalidJson')).toBeTrue();
     }));

  it('should validate jwksFile as valid if content is valid JSON',
     fakeAsync(() => {
       fixture.detectChanges();
       const jwksFileCtrl = component.securityConfigForm.get('jwksFile');

       const validJsonFile =
           new File(['{"keys": []}'], 'valid.json', {type: 'application/json'});

       const mockFileReader = {
         readAsText:
             jasmine.createSpy('readAsText').and.callFake(function(this: any) {
               this.result = '{"keys": []}';
               if (this.onload) this.onload();
             })
       };
       spyOn(window as any, 'FileReader').and.returnValue(mockFileReader);

       jwksFileCtrl?.setValue(validJsonFile);

       tick();  // Wait for async validator to complete

       expect(jwksFileCtrl?.hasError('invalidJson')).toBeFalse();
       expect(jwksFileCtrl?.valid).toBeTrue();
     }));

  it('should validate jwksFile with fileReadError if FileReader encounters an error',
     fakeAsync(() => {
       fixture.detectChanges();
       const jwksFileCtrl = component.securityConfigForm.get('jwksFile');

       const validJsonFile =
           new File(['{"keys": []}'], 'valid.json', {type: 'application/json'});

       const mockFileReader = {
         readAsText:
             jasmine.createSpy('readAsText').and.callFake(function(this: any) {
               if (this.onerror) this.onerror();
             })
       };
       spyOn(window as any, 'FileReader').and.returnValue(mockFileReader);

       jwksFileCtrl?.setValue(validJsonFile);

       tick();  // Wait for async validator to complete

       expect(jwksFileCtrl?.hasError('fileReadError')).toBeTrue();
     }));

  it('should handle onNextTab and onPreviousTab', () => {
    fixture.detectChanges();

    component.componentConfigTabs = {selectedIndex: 0} as any;
    component.totalInternalSteps = 3;

    component.onNextTab();
    expect(component.componentConfigTabs.selectedIndex).toBe(1);

    component.onPreviousTab();
    expect(component.componentConfigTabs.selectedIndex).toBe(0);
  });

  it('should navigate back from step 0 on onPreviousTab', () => {
    fixture.detectChanges();
    component.componentConfigTabs = {selectedIndex: 0} as any;

    component.onPreviousTab();
    expect(TestBed.inject(Router).navigate).toHaveBeenCalledWith([
      'installer', 'domain-configuration'
    ]);
  });

  it('should consider URLs with leading/trailing spaces as valid', () => {
    fixture.detectChanges();
    const registryUrlCtrl = component.registryConfigForm.get('registryUrl');

    registryUrlCtrl?.setValue('  http://example.com  ');
    fixture.detectChanges();

    expect(registryUrlCtrl?.valid).toBeTrue();
    expect(registryUrlCtrl?.hasError('pattern')).toBeFalse();
  });
});