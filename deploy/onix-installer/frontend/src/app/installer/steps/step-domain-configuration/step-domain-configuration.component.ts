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

import { Component, OnInit, OnDestroy, Output, EventEmitter, ChangeDetectionStrategy, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule, AbstractControl } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatRadioModule } from '@angular/material/radio';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatTabsModule } from '@angular/material/tabs';
import { MatTabChangeEvent } from '@angular/material/tabs';

import { Subject } from 'rxjs';
import { takeUntil, debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { Router } from '@angular/router';

import { InstallerStateService } from '../../../core/services/installer-state.service';
import { InstallerState, DomainProvider, DomainConfig, ComponentSubdomainPrefix, ComponentDomainKey, SubdomainConfig } from '../../types/installer.types';


@Component({
  selector: 'app-step-domain-config',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatRadioModule,
    MatTooltipModule,
    MatCheckboxModule,
    MatTabsModule
  ],
  templateUrl: './step-domain-configuration.component.html',
  styleUrls: ['./step-domain-configuration.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepDomainConfigComponent implements OnInit, OnDestroy {
  @Output() nextStep = new EventEmitter<void>();
  @Output() backStep = new EventEmitter<void>();

  domainConfigForm!: FormGroup;
  installerState!: InstallerState;

  currentInternalStep: number = 1;
  readonly totalInternalSteps: number = 3;

  componentsToConfigure: ComponentDomainKey[] = [];

  private unsubscribe$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private installerStateService: InstallerStateService,
    private router: Router,
    private cdr: ChangeDetectorRef
  ) { }

  ngOnInit(): void {
    this.domainConfigForm = this.fb.group({
      componentPrefixes: this.fb.array([]),

      globalDomainDetails: this.fb.group({
        domainType: [null as DomainProvider | null, Validators.required],
        baseDomain: ['', Validators.required],
        dnsZone: [''],
        actionRequiredAcknowledged: [false]
      }),
    });

    this.installerStateService.installerState$
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe(state => {
        this.installerState = state;
        this.initializeFormFromState(state);
        this.cdr.detectChanges();
      });

    this.domainConfigForm.get('globalDomainDetails.domainType')?.valueChanges.pipe(
      takeUntil(this.unsubscribe$),
      distinctUntilChanged()
    ).subscribe((value: DomainProvider) => {
      this.toggleGlobalDomainDetailsValidators(value);
      this.cdr.detectChanges();
    });
  }

  ngOnDestroy(): void {
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

  get componentPrefixes(): FormArray {
    return this.domainConfigForm.get('componentPrefixes') as FormArray;
  }

  get globalDomainDetailsFormGroup(): FormGroup {
    return this.domainConfigForm.get('globalDomainDetails') as FormGroup;
  }

  // --- Tab Enabling Logic ---
  isStep1Enabled(): boolean {
    return true; // Step 1 is always enabled
  }

  isStep2Enabled(): boolean {
    return this.componentPrefixes.valid;
  }

  isStep3Enabled(): boolean {
    return this.globalDomainDetailsFormGroup.get('domainType')?.valid || false;
  }

  isCurrentStepValid(): boolean {
    switch (this.currentInternalStep) {
      case 1:
        return this.componentPrefixes.valid;
      case 2:
        return this.globalDomainDetailsFormGroup.get('domainType')?.valid || false;
      case 3:
        return this.globalDomainDetailsFormGroup.valid;
      default:
        return false;
    }
  }

  private initializeFormFromState(state: InstallerState): void {
    this.componentsToConfigure = [];
    const deploymentGoal = state.deploymentGoal;

    if (deploymentGoal.all || deploymentGoal.registry) {
      this.componentsToConfigure.push('registry');
      this.componentsToConfigure.push('registry_admin');
    }
    if (deploymentGoal.all || deploymentGoal.gateway) {
      this.componentsToConfigure.push('gateway');
    }
    if (deploymentGoal.all || deploymentGoal.bap || deploymentGoal.bpp) {
        this.componentsToConfigure.push('adapter');
    }
    if (deploymentGoal.all || deploymentGoal.gateway || deploymentGoal.bap || deploymentGoal.bpp) {
        this.componentsToConfigure.push('subscriber');
    }

    while (this.componentPrefixes.length !== 0) {
      this.componentPrefixes.removeAt(0);
    }
    this.componentsToConfigure.forEach((compKey: ComponentDomainKey) => {
      const existingPrefixConfig = state.componentSubdomainPrefixes.find((c: ComponentSubdomainPrefix) => c.component === compKey);
      this.componentPrefixes.push(this.fb.group({
        component: [compKey, Validators.required],
        subdomainPrefix: [existingPrefixConfig?.subdomainPrefix || compKey, Validators.required]
      }));
    });

    if (state.globalDomainConfig) {
      this.domainConfigForm.get('globalDomainDetails')?.patchValue(state.globalDomainConfig, { emitEvent: false });
      if (state.globalDomainConfig.domainType) {
        this.toggleGlobalDomainDetailsValidators(state.globalDomainConfig.domainType);
      }
    }

    const globalIpControl = this.domainConfigForm.get('globalDomainDetails.globalIp');
    if (globalIpControl && state.appExternalIp) {
      globalIpControl.setValue(state.appExternalIp, { emitEvent: false });
    }

    this.componentPrefixes.controls.sort((a: AbstractControl, b: AbstractControl) => {
        const order = ['registry', 'registryAdmin', 'gateway', 'adapter', 'subscriber'];
        const compA = a instanceof FormGroup ? a.get('component')?.value : undefined;
        const compB = b instanceof FormGroup ? b.get('component')?.value : undefined;

        if (compA === undefined || compB === undefined) {
          return 0;
        }
        return order.indexOf(compA) - order.indexOf(compB);
    });
  }

  toggleGlobalDomainDetailsValidators(domainType: DomainProvider): void {
    const globalDomainDetailsGroup = this.domainConfigForm.get('globalDomainDetails') as FormGroup;
    const baseDomainControl = globalDomainDetailsGroup.get('baseDomain');
    const dnsZoneControl = globalDomainDetailsGroup.get('dnsZone');
    const actionRequiredControl = globalDomainDetailsGroup.get('actionRequiredAcknowledged');
    const globalIpControl = globalDomainDetailsGroup.get('globalIp');

    baseDomainControl?.clearValidators();
    globalIpControl?.clearValidators();
    globalIpControl?.setValue(null);

    if (domainType === 'google_domain') {
      baseDomainControl?.setValidators(Validators.required);
      dnsZoneControl?.setValidators(Validators.required);
      actionRequiredControl?.clearValidators();
      actionRequiredControl?.setValue(false);
    } else { // 'other_domain'
      dnsZoneControl?.clearValidators();
      dnsZoneControl?.setValue('');
      actionRequiredControl?.setValidators(Validators.requiredTrue);
    }

    baseDomainControl?.updateValueAndValidity();
    dnsZoneControl?.updateValueAndValidity();
    actionRequiredControl?.updateValueAndValidity();
    globalIpControl?.updateValueAndValidity();
  }

  onTabChange(event: MatTabChangeEvent): void {
    // Check if the target tab is enabled before allowing the switch
    // If not, revert to the current step (which is 1-indexed)
    if (event.index === 0 && !this.isStep1Enabled()) {
        this.currentInternalStep = 1;
    } else if (event.index === 1 && !this.isStep2Enabled() && this.currentInternalStep < 2) {
        this.currentInternalStep = 1;
    } else if (event.index === 2 && !this.isStep3Enabled() && this.currentInternalStep < 3) {
        this.currentInternalStep = 2;
    } else {
        this.currentInternalStep = event.index + 1;
    }
  }


  onNextInternal(): void {
    switch (this.currentInternalStep) {
      case 1:
        this.componentPrefixes.markAllAsTouched();
        if (this.componentPrefixes.invalid) {
          console.error('Component subdomain prefixes are invalid.');
          return;
        }

        setTimeout(() => {
          this.installerStateService.updateComponentSubdomainPrefixes(this.componentPrefixes.value);
        }, 0);
        break;

      case 2:
        const globalDomainTypeControl = this.domainConfigForm.get('globalDomainDetails.domainType');
        globalDomainTypeControl?.markAsTouched();
        if (globalDomainTypeControl?.invalid) {
          console.error('Global Domain Type is not selected.');
          return;
        }
        // Update state after UI transition to reduce glitching
        setTimeout(() => {
          const currentGlobalDomainDetails = this.domainConfigForm.get('globalDomainDetails')?.value as DomainConfig;
          this.installerStateService.updateGlobalDomainConfig(currentGlobalDomainDetails);
        }, 0);
        break;

      case 3:
        const globalDomainDetailsGroup = this.domainConfigForm.get('globalDomainDetails') as FormGroup;
        globalDomainDetailsGroup.markAllAsTouched();

        if (globalDomainDetailsGroup.invalid) {
          console.error('Domain details are invalid or action not acknowledged.');
          return;
        }
        const finalGlobalDomainDetails: DomainConfig = globalDomainDetailsGroup.value;
        const appExternalIp = this.installerState.appExternalIp;

        const finalSubdomainConfigs: SubdomainConfig[] = [];
        this.componentPrefixes.controls.forEach((control: AbstractControl) => {
          const componentPrefixData = control.value as ComponentSubdomainPrefix;
          const baseDomain = finalGlobalDomainDetails.domainType === 'google_domain' ? finalGlobalDomainDetails.baseDomain : (this.installerState.globalDomainConfig?.baseDomain || '');

          const fullDomainName = `${componentPrefixData.subdomainPrefix}`;

          const subdomainConfig: SubdomainConfig = {
            component: componentPrefixData.component,
            domainType: finalGlobalDomainDetails.domainType!,
            subdomainName: fullDomainName
          };

          if (finalGlobalDomainDetails.domainType === 'google_domain') {
            subdomainConfig.googleDomain = {
              domain: finalGlobalDomainDetails.baseDomain,
              zone: finalGlobalDomainDetails.dnsZone!
            };
          } else if (finalGlobalDomainDetails.domainType === 'other_domain') {
            subdomainConfig.customDomain = {
              globalIp: appExternalIp || ''
            };
          }
          finalSubdomainConfigs.push(subdomainConfig);
        });

        this.installerStateService.updateGlobalDomainConfig(finalGlobalDomainDetails);
        this.installerStateService.updateSubdomainConfigs(finalSubdomainConfigs);
         this.router.navigate(['installer', 'deploy-app']);
        return;
    }


    if (this.currentInternalStep < this.totalInternalSteps) {
      this.currentInternalStep++;
    }

    this.cdr.detectChanges();
  }


  onBackInternal(): void {
    if (this.currentInternalStep > 1) {
      this.currentInternalStep--;
    } else {
      this.router.navigate(['installer', 'deploy-infra']);
    }
  }

  getErrorMessage(control: AbstractControl | null, fieldName: string): string {
    if (!control || !control.touched) {
      return '';
    }
    if (control.hasError('required')) {
      return `${fieldName} is required.`;
    }
    if (control.hasError('requiredTrue')) {
      return `Please acknowledge this action to proceed.`;
    }
    return '';
  }
}