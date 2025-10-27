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

// src/app/installer/steps/step-subscribe/step-subscribe.component.ts

import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AbstractControl, FormBuilder, FormGroup, ReactiveFormsModule, ValidationErrors, ValidatorFn, Validators } from '@angular/forms';

import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner'; // Import MatProgressSpinnerModule

import { InstallerStateService } from '../../../core/services/installer-state.service';
import { ApiService } from '../../../core/services/api.service';
import { jsonValidator } from '../../../shared/validators/custom-validators';



@Component({
  selector: 'app-step-subscription',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './step-subscribe.html',
  styleUrl: './step-subscribe.css'
})
export class StepSubscribe implements OnInit {
  public subscriptionForm!: FormGroup;
  public subscriptionTypes: string[] = [];
  public subscriptionUrl = ""

  public showStatusPopup: boolean = false;
  public popupMessage: string = '';
  public showSpinner: boolean = false;
  public isError: boolean = false;
  public responseMessageId: string | null = null;
  public popupIcon: string = '';

  private componentUrlMap: { [key: string]: string } = {};

  constructor(
    private fb: FormBuilder,
    private router: Router,
    private installerStateService: InstallerStateService,
    private cdr: ChangeDetectorRef,
    private apiService: ApiService
  ) {
  }

  ngOnInit(): void {
    const currentState = this.installerStateService.getCurrentState();
    const deploymentGoal = currentState.deploymentGoal;
    const deployedServiceUrls = currentState.deployedServiceUrls;

    if (deploymentGoal) {
      this.subscriptionTypes = (Object.keys(deploymentGoal) as Array<keyof typeof deploymentGoal>)
        .filter(key => deploymentGoal[key] === true && key !== 'all')
        .map(key => {
          if (key === 'bap') return 'BAP';
          if (key === 'bpp') return 'BPP';
          if (key === 'gateway') return 'BG';
          return null;
        }).filter(Boolean) as string[];
    }

    this.subscriptionForm = this.fb.group({
      type: ['', Validators.required],
      subscriberId: ['', Validators.required],
      url: [{ value: '', disabled: true }, Validators.required],
      domain: [''],
      location: ['', jsonValidator()]
    });

    this.componentUrlMap = {
      'BAP': deployedServiceUrls['adapter_bapTxnReceiver'] || '',
      'BPP': deployedServiceUrls['adapter_bppTxnReceiver'] || '',
      'BG': deployedServiceUrls['gateway'] || ''
    };

    this.subscriptionUrl = deployedServiceUrls['subscriber'] || '';

    this.subscriptionForm.get('type')?.valueChanges.subscribe(type => {
      const domainControl = this.subscriptionForm.get('domain');
      const urlControl = this.subscriptionForm.get('url');

      if (urlControl) {
        const newUrl = this.componentUrlMap[type] || '';
        urlControl.setValue(newUrl);
      }

      if (domainControl) {
        if (type === 'BAP' || type === 'BPP') {
          domainControl.setValidators([Validators.required]);
        } else {
          domainControl.clearValidators();
        }
        domainControl.updateValueAndValidity();
      }
    });

    this.subscriptionForm.get('type')?.updateValueAndValidity({ emitEvent: true });
  }

  onSubscriptionSubmit(): void {
    this.subscriptionForm.markAllAsTouched();
    if (this.subscriptionForm.invalid) {
      return;
    }

    const formValue = this.subscriptionForm.value;

    const subscrptionPayload = {
      subscriber_id: formValue.subscriberId,
      type: formValue.type,
      domain: formValue.domain || "*",
      url: this.componentUrlMap[formValue.type],
      location: formValue.location ? JSON.parse(formValue.location) : null
    };

    const payload = {
      'targetUrl': `${this.subscriptionUrl}/subscribe`,
      'payload': subscrptionPayload
    }

    this.showStatusPopup = true;
    this.popupMessage = 'Submitting subscription request... please wait.';
    this.showSpinner = true;
    this.isError = false;
    this.responseMessageId = null;
    this.popupIcon = '';
    console.log('Subscription payload:', payload);
    this.apiService.subscribeToNetwork(payload).subscribe({
      next: (response: any) => {
        console.log('Subscription successful:', response);

        this.popupMessage = 'Subscription request sent successfully!';
        this.showSpinner = false;
        this.isError = false;
        this.popupIcon = 'check_circle';


        if (response) {
          this.responseMessageId = response
        }

        this.subscriptionForm.reset();
        this.subscriptionForm.get('url')?.disable();
        this.subscriptionForm.markAsUntouched();
        this.subscriptionForm.markAsPristine();
      },
      error: (error) => {
        console.error('Subscription failed:', error);
        this.popupMessage = `Subscription failed: ${error.error?.message || error.message || 'Unknown error'}`;
        this.showSpinner = false;
        this.isError = true;
        this.popupIcon = 'error_outline';
        this.responseMessageId = null;

        this.showStatusPopup = true;
      }
    });
  }

  closePopupAndNavigate(): void {
    this.showStatusPopup = false;
  }

  onBack(): void {
    this.router.navigate(['installer', 'health-checks']);
  }
}

