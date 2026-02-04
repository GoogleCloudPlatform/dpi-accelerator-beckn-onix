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

import {ChangeDetectorRef} from '@angular/core';
import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {ReactiveFormsModule} from '@angular/forms';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {BehaviorSubject, EMPTY, of, Subject, throwError} from 'rxjs';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {GcpConfiguration} from '../../types/installer.types';

import {StepGcpConnectionComponent} from './step-gcp-connection.component';

// Initialize the Angular testing environment.
getTestBed().initTestEnvironment(
    BrowserDynamicTestingModule,
    platformBrowserDynamicTesting(),
);

const mockProjects = ['gcp-project-1', 'gcp-project-2', 'another-project'];
const mockRegions = ['us-central1', 'us-east1', 'europe-west1'];

class MockInstallerStateService {
  private state =
      new BehaviorSubject<{gcpConfiguration: GcpConfiguration | null}>(
          {gcpConfiguration: null});

  getCurrentState() {
    return this.state.getValue();
  }

  updateGcpConfiguration(gcpConfig: GcpConfiguration) {
    this.state.next({...this.state.getValue(), gcpConfiguration: gcpConfig});
  }

  setInitialState(initialState: any) {
    this.state.next(initialState);
  }
}

class MockApiService {
  getGcpProjectNames() {
    return of(mockProjects);
  }

  getGcpRegions() {
    return of(mockRegions);
  }
}

describe('StepGcpConnectionComponent', () => {
  let component: StepGcpConnectionComponent;
  let fixture: ComponentFixture<StepGcpConnectionComponent>;
  let router: Router;
  let installerStateService: MockInstallerStateService;
  let apiService: MockApiService;
  let cdr: ChangeDetectorRef;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [
            ReactiveFormsModule, RouterTestingModule.withRoutes([]),
            NoopAnimationsModule, StepGcpConnectionComponent
          ],
          providers: [
            {
              provide: InstallerStateService,
              useClass: MockInstallerStateService
            },
            {provide: ApiService, useClass: MockApiService},
            ChangeDetectorRef,
          ],
        })
        .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(StepGcpConnectionComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    installerStateService = TestBed.inject(InstallerStateService) as unknown as
        MockInstallerStateService;
    apiService = TestBed.inject(ApiService) as unknown as MockApiService;
    cdr = fixture.debugElement.injector.get(ChangeDetectorRef);
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  describe('Initialization and API Fetching', () => {
    it('should initialize the form and fetch projects and regions on init',
       fakeAsync(() => {
         const fetchProjectsSpy =
             spyOn(apiService, 'getGcpProjectNames').and.callThrough();
         const fetchRegionsSpy =
             spyOn(apiService, 'getGcpRegions').and.callThrough();

         fixture.detectChanges();
         tick();

         expect(component.gcpConnectionForm).toBeDefined();
         expect(fetchProjectsSpy).toHaveBeenCalled();
         expect(fetchRegionsSpy).toHaveBeenCalled();
         expect(component.gcpProjects).toEqual(mockProjects);
         expect(component.gcpRegions).toEqual(mockRegions);
         expect(component.isProjectsLoading).toBeFalse();
         expect(component.isRegionsLoading).toBeFalse();
       }));

    it('should initialize form with values from installer state',
       fakeAsync(() => {
         const initialState = {
           gcpConfiguration: {projectId: 'gcp-project-1', region: 'us-east1'}
         };
         installerStateService.setInitialState(initialState);

         fixture.detectChanges();
         tick();

         expect(component.gcpConnectionForm.value)
             .toEqual(initialState.gcpConfiguration);
       }));
  });

  describe('Form Interaction and Filtering', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('should filter projects based on search input', fakeAsync(() => {
         component.projectSearchCtrl.setValue('gcp');
         tick(300);
         expect(component.filteredGcpProjects.value).toEqual([
           'gcp-project-1', 'gcp-project-2'
         ]);

         component.projectSearchCtrl.setValue('another');
         tick(300);
         expect(component.filteredGcpProjects.value).toEqual([
           'another-project'
         ]);
       }));

    it('should filter regions based on search input', fakeAsync(() => {
         component.regionSearchCtrl.setValue('us-');
         tick(300);
         expect(component.filteredGcpRegions.value).toEqual([
           'us-central1', 'us-east1'
         ]);

         component.regionSearchCtrl.setValue('europe');
         tick(300);
         expect(component.filteredGcpRegions.value).toEqual(['europe-west1']);
       }));

    it('should update installer state when form value changes', () => {
      const updateStateSpy =
          spyOn(installerStateService, 'updateGcpConfiguration');
      const testData = {projectId: 'gcp-project-1', region: 'us-central1'};

      component.gcpConnectionForm.setValue(testData);

      expect(updateStateSpy).toHaveBeenCalledWith(testData);
    });
  });

  describe('Navigation', () => {
    beforeEach(fakeAsync(() => {
      fixture.detectChanges();
      tick();
    }));

    it('should not navigate on "Next" if the form is invalid', () => {
      const navigateSpy = spyOn(router, 'navigate');
      const emitSpy = spyOn(component.nextStep, 'emit');

      component.onNext();

      expect(navigateSpy).not.toHaveBeenCalled();
      expect(emitSpy).not.toHaveBeenCalled();
    });

    it('should emit nextStep and navigate on "Next" if the form is valid',
       () => {
         const navigateSpy = spyOn(router, 'navigate');
         const emitSpy = spyOn(component.nextStep, 'emit');

         component.gcpConnectionForm.setValue(
             {projectId: 'gcp-project-1', region: 'us-central1'});

         component.onNext();

         expect(emitSpy).toHaveBeenCalled();
         expect(navigateSpy).toHaveBeenCalledWith([
           'installer', 'deploy-infra'
         ]);
       });

    it('should emit previousStep and navigate on "Back"', () => {
      const navigateSpy = spyOn(router, 'navigate');
      const emitSpy = spyOn(component.previousStep, 'emit');

      component.onBack();

      expect(emitSpy).toHaveBeenCalled();
      expect(navigateSpy).toHaveBeenCalledWith(['installer', 'prerequisites']);
    });
  });

  describe('Lifecycle Hooks', () => {
    it('should complete the unsubscribe subject on ngOnDestroy', () => {
      const unsubscribeNextSpy = spyOn(component['unsubscribe$'], 'next');
      const unsubscribeCompleteSpy =
          spyOn(component['unsubscribe$'], 'complete');

      component.ngOnDestroy();

      expect(unsubscribeNextSpy).toHaveBeenCalled();
      expect(unsubscribeCompleteSpy).toHaveBeenCalled();
    });
  });
});