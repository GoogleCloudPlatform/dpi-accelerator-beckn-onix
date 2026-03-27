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

import {getTestBed, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';
import {take} from 'rxjs/operators';

import {InstallerState} from '../../installer/types/installer.types';

import {ApiService} from './api.service';
import {InstallerStateService} from './installer-state.service';

// Prevent js_scrub from stripping the imports
const _dummyInstallerStateService = InstallerStateService;
const _dummyApiService = ApiService;
const _dummyTakeInstaller = take;
const _dummyOf = of;

describe('InstallerStateService', () => {
  let service: InstallerStateService;
  let apiServiceMock: jasmine.SpyObj<ApiService>;

  beforeEach(() => {
    apiServiceMock = jasmine.createSpyObj(
        'ApiService',
        ['getState', 'getInstallerState', 'storeState', 'storeBulkState']);
    apiServiceMock.getState.and.returnValue(of({} as InstallerState));
    apiServiceMock.getInstallerState.and.returnValue(of({} as InstallerState));
    apiServiceMock.storeState.and.returnValue(of({} as any));
    apiServiceMock.storeBulkState.and.returnValue(of({} as any));

    TestBed.configureTestingModule({
      providers: [
        InstallerStateService,
        {provide: ApiService, useValue: apiServiceMock},
      ],
    });
    service = TestBed.inject(InstallerStateService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should update current step', (done) => {
    service.updateCurrentStep(2);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.currentStepIndex).toBe(2);
      done();
    });
  });

  it('should update prerequisites met', (done) => {
    service.updatePrerequisitesMet(true);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.prerequisitesMet).toBe(true);
      done();
    });
  });

  it('should update GCP configuration', (done) => {
    const config = {projectId: 'p1', region: 'r1', credentialsPath: 'path'};
    service.updateGcpConfiguration(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.gcpConfiguration).toEqual(config as any);
      done();
    });
  });

  it('should update global domain config', (done) => {
    const config = {domainName: 'd1'};
    service.updateGlobalDomainConfig(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.globalDomainConfig).toEqual(config as any);
      done();
    });
  });

  it('should update component subdomain prefixes', (done) => {
    const prefixes = [{componentName: 'c1', prefix: 'p1'}];
    service.updateComponentSubdomainPrefixes(prefixes as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.componentSubdomainPrefixes).toEqual(prefixes as any);
      done();
    });
  });

  it('should update subdomain configs', (done) => {
    const configs = [{domainName: 'd1'}];
    service.updateSubdomainConfigs(configs as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.subdomainConfigs).toEqual(configs as any);
      done();
    });
  });

  it('should update docker image configs', (done) => {
    const configs = [{imageName: 'i1'}];
    service.updateDockerImageConfigs(configs as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.dockerImageConfigs).toEqual(configs as any);
      done();
    });
  });

  it('should update app specific configs', (done) => {
    const configs = [{configName: 'c1'}];
    service.updateAppSpecificConfigs(configs as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.appSpecificConfigs).toEqual(configs as any);
      done();
    });
  });

  it('should update health check statuses', (done) => {
    const statuses = [{componentName: 'c1', status: 'ok'}];
    service.updateHealthCheckStatuses(statuses as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.healthCheckStatuses).toEqual(statuses as any);
      done();
    });
  });

  it('should update app deploy image config', (done) => {
    const config = {registryImageUrl: 'url1'};
    service.updateAppDeployImageConfig(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.appDeployImageConfig)
          .toEqual(jasmine.objectContaining(config) as any);
      done();
    });
  });

  it('should update app deploy registry config', (done) => {
    const config = {registryUrl: 'url2'};
    service.updateAppDeployRegistryConfig(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.appDeployRegistryConfig)
          .toEqual(jasmine.objectContaining(config) as any);
      done();
    });
  });

  it('should update app deploy gateway config', (done) => {
    const config = {gatewaySubscriptionId: 'id1'};
    service.updateAppDeployGatewayConfig(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.appDeployGatewayConfig).toEqual(config as any);
      done();
    });
  });

  it('should update app deploy adapter config', (done) => {
    const config = {enableSchemaValidation: true};
    service.updateAppDeployAdapterConfig(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.appDeployAdapterConfig).toEqual(config as any);
      done();
    });
  });

  it('should update app deploy security config', (done) => {
    const config = {enableInBoundAuth: true};
    service.updateAppDeploySecurityConfig(config as any);
    service.installerState$.pipe(take(1)).subscribe((state) => {
      expect(state.appDeploySecurityConfig)
          .toEqual(jasmine.objectContaining(config) as any);
      done();
    });
  });

  it('should set deployment state (isDeploying)', (done) => {
    service.setDeploymentState(true);
    service.isDeploying$.pipe(take(1)).subscribe((isDeploying) => {
      expect(isDeploying).toBe(true);
      done();
    });
  });
});
