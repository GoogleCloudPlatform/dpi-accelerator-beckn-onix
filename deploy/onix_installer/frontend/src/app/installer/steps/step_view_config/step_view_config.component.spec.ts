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

import {HttpClient} from '@angular/common/http';
import {ComponentFixture, getTestBed, TestBed} from '@angular/core/testing';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {of} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {InstallerState} from '../../types/installer.types';

import {StepViewConfigComponent} from './step_view_config.component';

describe('StepViewConfigComponent', () => {
  let httpClientSpy: jasmine.SpyObj<HttpClient>;
  let installerStateServiceSpy: jasmine.SpyObj<InstallerStateService>;
  let component: StepViewConfigComponent;
  let fixture: ComponentFixture<StepViewConfigComponent>;

  beforeEach(async () => {
    httpClientSpy = jasmine.createSpyObj('HttpClient', ['get', 'post']);
    installerStateServiceSpy = jasmine.createSpyObj(
      'InstallerStateService',
      ['getCurrentState'],
    );

    httpClientSpy.get.and.returnValue(of({files: []}));
    httpClientSpy.post.and.returnValue(of({}));
    installerStateServiceSpy.getCurrentState.and.returnValue({
      appName: 'test-app',
      deploymentGoal: {
        bap: false,
        bpp: false,
        gateway: false,
        registry: false,
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
    } as InstallerState);

    await TestBed
        .configureTestingModule({
          imports: [StepViewConfigComponent],
          providers: [
            {provide: HttpClient, useValue: httpClientSpy},
            {
              provide: InstallerStateService,
              useValue: installerStateServiceSpy
            },
          ],
        })
        .compileComponents();

    fixture = TestBed.createComponent(StepViewConfigComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
