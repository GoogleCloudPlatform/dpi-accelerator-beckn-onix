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

import {ComponentFixture, getTestBed, TestBed} from '@angular/core/testing';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {of} from 'rxjs';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {InstallerState} from '../../types/installer.types';

import {StepViewConfigComponent} from './step_view_config.component';

describe('StepViewConfigComponent', () => {
  let apiServiceSpy: jasmine.SpyObj<ApiService>;
  let installerStateServiceSpy: jasmine.SpyObj<InstallerStateService>;
  let component: StepViewConfigComponent;
  let fixture: ComponentFixture<StepViewConfigComponent>;

  beforeEach(async () => {
    apiServiceSpy = jasmine.createSpyObj(
        'ApiService', ['getConfigPaths', 'getConfigData', 'updateConfigData']);
    installerStateServiceSpy = jasmine.createSpyObj(
        'InstallerStateService',
        ['getCurrentState', 'updateState'],
    );

    apiServiceSpy.getConfigPaths.and.returnValue(of({files: []}));
    apiServiceSpy.getConfigData.and.returnValue(of({content: ''}));
    apiServiceSpy.updateConfigData.and.returnValue(of({}));
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
      enableCloudArmor: false,
      cloudArmorRateLimit: 100,
    } as InstallerState);

    await TestBed
        .configureTestingModule({
          imports: [StepViewConfigComponent],
          providers: [
            {provide: ApiService, useValue: apiServiceSpy},
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

  it('should fetch and group file paths on init', () => {
    apiServiceSpy.getConfigPaths.and.returnValue(
        of({files: ['root.yaml', 'Folder/child.yaml']}));
    component.ngOnInit();
    expect(apiServiceSpy.getConfigPaths).toHaveBeenCalled();
    expect(component.files.length).toBe(3);  // 1 root, 1 folder, 1 child
    const folder = component.files.find(f => f.type === 'folder');
    expect(folder?.name).toBe('Folder');
  });

  it('should load file content when editing a file', () => {
    const file = {path: 'root.yaml', name: 'root.yaml', type: 'file'};
    apiServiceSpy.getConfigData.and.returnValue(of({content: 'test content'}));

    component.onEditFile(file);

    expect(component.currentFile).toBe(file);
    expect(component.isEditing).toBeTrue();
    expect(component.fileContent).toBe('test content');
    expect(component.originalFileContent).toBe('test content');
    expect(component.isLoading).toBeFalse();
  });

  it('should save file content and check YAML validity', () => {
    component.currentFile = {path: 'root.yaml'};
    component.fileContent = 'key: value';
    component.originalFileContent = 'key: old_value';

    component.onSave();

    expect(component.validationError).toBeNull();
    expect(installerStateServiceSpy.updateState).toHaveBeenCalledWith({
      isConfigChanged: true
    });
    expect(apiServiceSpy.updateConfigData)
        .toHaveBeenCalledWith({path: 'root.yaml', content: 'key: value'});
    expect(component.isSaving).toBeFalse();
    expect(component.isEditing).toBeFalse();
  });

  it('should show validation error on invalid YAML', () => {
    component.currentFile = {path: 'root.yaml'};
    component.fileContent = 'invalid:\n  - yaml\n  : here';

    component.onSave();

    expect(component.validationError).toContain('Invalid YAML');
    expect(apiServiceSpy.updateConfigData).not.toHaveBeenCalled();
  });

  it('should show validation error on missing value in YAML', () => {
    component.currentFile = {path: 'root.yaml'};
    component.fileContent = 'key: null\nother_key: value';

    component.onSave();

    expect(component.validationError)
        .toContain('The value for \'key\' cannot be empty.');
    expect(apiServiceSpy.updateConfigData).not.toHaveBeenCalled();
  });
});
