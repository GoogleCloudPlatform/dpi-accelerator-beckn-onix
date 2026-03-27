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
import {Router} from '@angular/router';
import {of} from 'rxjs';

import {InstallerStateService} from '../services/installer-state.service';

import {InstallerStepGuard} from './installer-step.guard';

function preventMerge() {}  // Prevent formatter from merging imports

describe('InstallerStepGuard', () => {
  let guard: InstallerStepGuard;
  let mockInstallerStateService: jasmine.SpyObj<InstallerStateService>;
  let mockRouter: jasmine.SpyObj<Router>;

  beforeEach(() => {
    mockInstallerStateService =
        jasmine.createSpyObj('InstallerStateService', ['getCurrentState']);
    mockRouter = jasmine.createSpyObj('Router', ['createUrlTree']);

    TestBed.configureTestingModule({
      providers: [
        InstallerStepGuard,
        {provide: InstallerStateService, useValue: mockInstallerStateService},
        {provide: Router, useValue: mockRouter},
      ],
    });

    guard = TestBed.inject(InstallerStepGuard);
  });

  it('should be created', () => {
    expect(guard).toBeTruthy();
  });

  it('should allow navigation if target step is less than highestStepReached',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 5,
       } as any);

       const mockRoute = {url: [{path: 'welcome'}]} as any;

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBeTrue();
         done();
       });
     });

  it('should redirect to goal if goal is missing for prerequisites', (done) => {
    mockInstallerStateService.getCurrentState.and.returnValue({
      highestStepReached: 0,
      installerGoal: null,
    } as any);

    const mockRoute = {url: [{path: 'prerequisites'}]} as any;
    const mockUrlTree = {} as any;
    mockRouter.createUrlTree.and.returnValue(mockUrlTree);

    guard.canActivate(mockRoute).subscribe((result) => {
      expect(result).toBe(mockUrlTree);
      expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
        '/installer/goal'
      ]);
      done();
    });
  });

  it('should redirect to prerequisites if prerequisites not met for gcp-connection',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         prerequisitesMet: false,
       } as any);

       const mockRoute = {url: [{path: 'gcp-connection'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/prerequisites'
         ]);
         done();
       });
     });

  it('should redirect to gcp-connection if gcp project is missing for deploy-infra',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         gcpConfiguration: {projectId: '', region: 'us-central1'},
       } as any);

       const mockRoute = {url: [{path: 'deploy-infra'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/gcp-connection'
         ]);
         done();
       });
     });

  it('should redirect to deploy-infra if deployment status not completed for domain-configuration',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         deploymentStatus: 'in-progress',
       } as any);

       const mockRoute = {url: [{path: 'domain-configuration'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/deploy-infra'
         ]);
         done();
       });
     });

  it('should redirect to domain-configuration if domain config missing for app-config',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         globalDomainConfig: null,
         subdomainConfigs: [],
       } as any);

       const mockRoute = {url: [{path: 'app-config'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/domain-configuration'
         ]);
         done();
       });
     });

  it('should redirect to app-config if app config invalid for view-config',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         isAppConfigValid: false,
       } as any);

       const mockRoute = {url: [{path: 'view-config'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/app-config'
         ]);
         done();
       });
     });

  it('should redirect to view-config if app deployment status not completed for view-deployment',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         appDeploymentStatus: 'in-progress',
         deployedServiceUrls: {},
       } as any);

       const mockRoute = {url: [{path: 'view-deployment'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/view-config'
         ]);
         done();
       });
     });

  it('should redirect to view-deployment if app deployment status not completed for health-checks',
     (done) => {
       mockInstallerStateService.getCurrentState.and.returnValue({
         highestStepReached: 0,
         appDeploymentStatus: 'in-progress',
         deployedServiceUrls: {},
       } as any);

       const mockRoute = {url: [{path: 'health-checks'}]} as any;
       const mockUrlTree = {} as any;
       mockRouter.createUrlTree.and.returnValue(mockUrlTree);

       guard.canActivate(mockRoute).subscribe((result) => {
         expect(result).toBe(mockUrlTree);
         expect(mockRouter.createUrlTree).toHaveBeenCalledWith([
           '/installer/view-deployment'
         ]);
         done();
       });
     });
});
