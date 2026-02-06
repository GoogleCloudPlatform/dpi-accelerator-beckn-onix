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
import {BehaviorSubject, of, Subscription} from 'rxjs';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {InstallerConstants} from '../../constants/installer-constants';

import {StepPrerequisitesComponent} from './step-prerequisites.component';

const mockPrerequisitesList = [
  'You have admin access to the GCP project.',
  'You have enabled the required APIs.',
  'You have configured billing for the project.',
];
InstallerConstants.PREREQUISITES_LIST = mockPrerequisitesList;

class MockInstallerStateService {
  private state = new BehaviorSubject<any>({prerequisitesMet: false});

  getCurrentState() {
    return this.state.getValue();
  }

  updatePrerequisitesMet(isMet: boolean) {
    this.state.next({...this.state.getValue(), prerequisitesMet: isMet});
  }

  setInitialState(initialState: any) {
    this.state.next(initialState);
  }
}

describe('StepPrerequisitesComponent', () => {
  let component: StepPrerequisitesComponent;
  let fixture: ComponentFixture<StepPrerequisitesComponent>;
  let router: Router;
  let installerStateService: MockInstallerStateService;
  let cdr: ChangeDetectorRef;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [
            ReactiveFormsModule, RouterTestingModule.withRoutes([]),
            NoopAnimationsModule, StepPrerequisitesComponent
          ],
          providers: [
            {
              provide: InstallerStateService,
              useClass: MockInstallerStateService
            },
            ChangeDetectorRef,
          ],
        })
        .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(StepPrerequisitesComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    // Get the instance of the mock service
    installerStateService = TestBed.inject(InstallerStateService) as unknown as
        MockInstallerStateService;
    cdr = TestBed.inject(ChangeDetectorRef);
    // Initial data binding and ngOnInit call
    fixture.detectChanges();
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  describe('Component Initialization (ngOnInit)', () => {
    it('should initialize the form with unchecked boxes if state is initially false',
       () => {
         const formArray = component.prerequisitesForm.get('items');
         expect(formArray).toBeDefined();
         expect(formArray?.value.length).toBe(mockPrerequisitesList.length);
         expect(formArray?.value.every((val: boolean) => !val)).toBeTrue();
         expect(component.allPrerequisitesMet).toBeTrue();
       });

    it('should initialize the form with all boxes checked if state is initially true',
       () => {
         installerStateService.setInitialState({prerequisitesMet: true});
         component.ngOnInit();
         fixture.detectChanges();

         const formArray = component.prerequisitesForm.get('items');
         expect(formArray?.value.every((val: boolean) => val)).toBeTrue();
         expect(component.allPrerequisitesMet).toBeTrue();
       });

    it('should subscribe to form value changes and call updatePrerequisitesState',
       fakeAsync(() => {
         const updateSpy = spyOn(component as any, 'updatePrerequisitesState')
                               .and.callThrough();

         const checkbox = component.getCheckboxControl(0);
         checkbox.setValue(true);

         tick();
         expect(updateSpy).toHaveBeenCalledTimes(1);
       }));
  });

  describe('Form Validation and State', () => {
    it('should be invalid if not all checkboxes are checked', fakeAsync(() => {
         component.getCheckboxControl(0).setValue(true);
         component.getCheckboxControl(1).setValue(true);
         component.getCheckboxControl(2).setValue(false);
         tick();

         expect(component.prerequisitesForm.valid).toBeFalse();
         expect(component.allPrerequisitesMet).toBeFalse();
       }));

    it('should be valid when all checkboxes are checked', fakeAsync(() => {
         mockPrerequisitesList.forEach((_, index) => {
           component.getCheckboxControl(index).setValue(true);
         });
         tick();

         expect(component.prerequisitesForm.valid).toBeTrue();
         expect(component.allPrerequisitesMet).toBeTrue();
       }));

    it('should call updatePrerequisitesMet on the service when state changes',
       fakeAsync(() => {
         const serviceSpy =
             spyOn(installerStateService, 'updatePrerequisitesMet')
                 .and.callThrough();

         mockPrerequisitesList.forEach((_, index) => {
           component.getCheckboxControl(index).setValue(true);
           tick();
         });
         expect(serviceSpy).toHaveBeenCalledWith(true);
       }));

    it('getCheckboxControl should return the correct form control', () => {
      const control = component.getCheckboxControl(1);
      expect(control).toBeDefined();
      expect(control.value).toBeFalse();
    });
  });

  describe('Navigation', () => {
    it('onNext should navigate to gcp-connection if form is valid',
       fakeAsync(() => {
         const navigateSpy = spyOn(router, 'navigate');

         mockPrerequisitesList.forEach((_, index) => {
           component.getCheckboxControl(index).setValue(true);
         });
         tick();

         component.onNext();

         expect(navigateSpy).toHaveBeenCalledWith([
           'installer', 'gcp-connection'
         ]);
       }));

    it('onNext should not navigate and should mark form as touched if invalid',
       () => {
         const navigateSpy = spyOn(router, 'navigate');
         const markTouchedSpy =
             spyOn(component.prerequisitesForm, 'markAllAsTouched')
                 .and.callThrough();

         component.getCheckboxControl(0).setValue(true);
         component.onNext();

         expect(markTouchedSpy).toHaveBeenCalled();
         expect(navigateSpy).not.toHaveBeenCalled();
       });

    it('onBack should navigate to goal', () => {
      const navigateSpy = spyOn(router, 'navigate');
      component.onBack();
      expect(navigateSpy).toHaveBeenCalledWith(['installer', 'goal']);
    });
  });

  describe('Component Destruction (ngOnDestroy)', () => {
    it('should unsubscribe from the form subscription', () => {
      const formSubscription = component['formSubscription'] as Subscription;
      const unsubscribeSpy = spyOn(formSubscription, 'unsubscribe');
      component.ngOnDestroy();

      expect(unsubscribeSpy).toHaveBeenCalled();
    });
  });
});