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
import {AbstractControl, FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatSelectModule} from '@angular/material/select';
import {By} from '@angular/platform-browser';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {Router} from '@angular/router';
import {of, throwError} from 'rxjs';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {jsonValidator} from '../../../shared/validators/custom-validators';

import {StepSubscribe} from './step-subscribe';

const mockRouter = {
  navigate: jasmine.createSpy('navigate')
};

const mockInstallerStateService = {
  getCurrentState: () => ({
    deploymentGoal: {all: false, bap: true, bpp: true, gateway: true},

    deployedServiceUrls: {
      'subscriber': 'https://subscriber.beckn.org',
      'adapter_bapTxnReceiver': 'https://adapter-bap.beckn.org',
      'adapter_bppTxnReceiver': 'https://adapter-bpp.beckn.org',
      'gateway': 'https://gateway.beckn.org'
    }
  })
};

const mockApiService = {
  subscribeToNetwork: jasmine.createSpy('subscribeToNetwork')
};


describe('StepSubscribe', () => {
  let component: StepSubscribe;
  let fixture: ComponentFixture<StepSubscribe>;
  let apiService: ApiService;
  let router: Router;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [
            StepSubscribe, ReactiveFormsModule, NoopAnimationsModule,
            MatSelectModule, MatFormFieldModule, MatInputModule,
            MatButtonModule, MatIconModule, MatProgressSpinnerModule
          ],
          providers: [
            FormBuilder, {provide: Router, useValue: mockRouter}, {
              provide: InstallerStateService,
              useValue: mockInstallerStateService
            },
            {provide: ApiService, useValue: mockApiService}
          ]
        })
        .compileComponents();

    fixture = TestBed.createComponent(StepSubscribe);
    component = fixture.componentInstance;
    apiService = TestBed.inject(ApiService);  // Get instance for spy checks
    router = TestBed.inject(Router);

    fixture.detectChanges();
  });

  afterEach(() => {
    mockRouter.navigate.calls.reset();
    mockApiService.subscribeToNetwork.calls.reset();
  });


  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  describe('ngOnInit', () => {
    it('should initialize the subscription form with all controls', () => {
      expect(component.subscriptionForm).toBeDefined();
      expect(component.subscriptionForm.get('type')).toBeDefined();
      expect(component.subscriptionForm.get('subscriberId')).toBeDefined();
      expect(component.subscriptionForm.get('url')).toBeDefined();
      expect(component.subscriptionForm.get('domain')).toBeDefined();
      expect(component.subscriptionForm.get('location')).toBeDefined();
    });

    it('should populate subscriptionTypes from installer state', () => {
      expect(component.subscriptionTypes).toEqual(['BAP', 'BPP', 'BG']);
    });

    it('should set the URL control to be disabled initially', () => {
      expect(component.subscriptionForm.get('url')?.disabled).toBeTrue();
    });

    it('should update URL and add domain validator when type changes to BAP',
       () => {
         const typeControl = component.subscriptionForm.get('type');
         const domainControl = component.subscriptionForm.get('domain');
         const urlControl = component.subscriptionForm.get('url');

         typeControl?.setValue('BAP');
         fixture.detectChanges();

         expect(urlControl?.value).toBe('https://adapter-bap.beckn.org');
         expect(domainControl?.hasValidator(Validators.required)).toBeTrue();
       });

    it('should update URL and remove domain validator when type changes to BG (Gateway)',
       () => {
         const typeControl = component.subscriptionForm.get('type');
         const domainControl = component.subscriptionForm.get('domain');
         const urlControl = component.subscriptionForm.get('url');
         typeControl?.setValue('BAP');
         fixture.detectChanges();
         expect(domainControl?.hasValidator(Validators.required)).toBeTrue();
         typeControl?.setValue('BG');
         fixture.detectChanges();

         expect(urlControl?.value).toBe('https://gateway.beckn.org');
         expect(domainControl?.hasValidator(Validators.required)).toBeFalse();
       });
  });

  describe('onSubscriptionSubmit', () => {
    beforeEach(() => {
      component.subscriptionForm.setValue({
        type: 'BAP',
        subscriberId: 'test-bap-subscriber',
        url: '',
        domain: 'retail',
        location: '{"country": {"code": "IND"}}'
      });
      component.subscriptionForm.get('type')?.updateValueAndValidity(
          {emitEvent: true});
      fixture.detectChanges();
    });

    it('should not call the API if the form is marked invalid', () => {
      component.subscriptionForm.get('subscriberId')?.setValue('');
      fixture.detectChanges();

      component.onSubscriptionSubmit();
      expect(apiService.subscribeToNetwork).not.toHaveBeenCalled();
    });

    it('should call the API with the correct payload when form is valid',
       () => {
         mockApiService.subscribeToNetwork.and.returnValue(of('12345'));
         component.onSubscriptionSubmit();

         const expectedPayload = {
           targetUrl: 'https://subscriber.beckn.org/subscribe',
           payload: {
             subscriber_id: 'test-bap-subscriber',
             type: 'BAP',
             domain: 'retail',
             url: 'https://adapter-bap.beckn.org',
             location: {country: {code: 'IND'}}
           }
         };

         expect(apiService.subscribeToNetwork)
             .toHaveBeenCalledWith(expectedPayload);
       });

    it('should show a success popup and reset the form on successful subscription',
       fakeAsync(() => {
         mockApiService.subscribeToNetwork.and.returnValue(
             of({messageId: 'ack-123'}));

         component.onSubscriptionSubmit();
         tick();
         fixture.detectChanges();

         expect(component.showStatusPopup).toBeTrue();
         expect(component.isError).toBeFalse();
         expect(component.popupMessage)
             .toBe('Subscription request sent successfully!');
         expect(component.popupIcon).toBe('check_circle');

         expect(component.subscriptionForm.get('subscriberId')?.value)
             .toBeNull();
         expect(component.subscriptionForm.pristine).toBeTrue();
       }));

    it('should show an error popup on failed subscription', fakeAsync(() => {
         const errorResponse = {error: {message: 'Gateway timeout'}};
         mockApiService.subscribeToNetwork.and.returnValue(
             throwError(() => errorResponse));

         component.onSubscriptionSubmit();
         tick();
         fixture.detectChanges();

         expect(component.showStatusPopup).toBeTrue();
         expect(component.isError).toBeTrue();
         expect(component.popupMessage).toContain('Gateway timeout');
         expect(component.popupIcon).toBe('error_outline');
       }));
  });

  describe('Navigation and UI', () => {
    it('should navigate to health-checks page onBack()', () => {
      component.onBack();
      expect(router.navigate).toHaveBeenCalledWith([
        'installer', 'health-checks'
      ]);
    });

    it('should hide the popup when closePopupAndNavigate() is called', () => {
      component.showStatusPopup = true;
      component.closePopupAndNavigate();
      expect(component.showStatusPopup).toBeFalse();
    });
  });
});


describe('jsonValidator', () => {
  const validator = jsonValidator();

  it('should return null for valid JSON', () => {
    const control = {value: '{ "key": "value" }'} as AbstractControl;
    expect(validator(control)).toBeNull();
  });

  it('should return { jsonInvalid: true } for invalid JSON', () => {
    const control = {value: '{ key: "value" }'} as AbstractControl;
    expect(validator(control)).toEqual({jsonInvalid: true});
  });

  it('should return null for an empty string or null value (to be handled by `required` validator)',
     () => {
       let control = {value: ''} as AbstractControl;
       expect(validator(control)).toBeNull();

       control = {value: null} as AbstractControl;
       expect(validator(control)).toBeNull();

       control = {value: '   '} as AbstractControl;
       expect(validator(control)).toBeNull();
     });
});