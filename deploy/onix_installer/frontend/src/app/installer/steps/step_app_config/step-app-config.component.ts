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

import {Clipboard, ClipboardModule} from '@angular/cdk/clipboard';
import {CommonModule} from '@angular/common';
import {ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, EventEmitter, Input, OnDestroy, OnInit, Output, ViewChild} from '@angular/core';
import {AbstractControl, AsyncValidatorFn, FormBuilder, FormControl, FormGroup, ReactiveFormsModule, ValidationErrors, Validators} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatRadioModule} from '@angular/material/radio';
import {MatSlideToggleModule} from '@angular/material/slide-toggle';
import {MatSnackBar, MatSnackBarModule} from '@angular/material/snack-bar';
import {MatTabGroup, MatTabsModule} from '@angular/material/tabs';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Router} from '@angular/router';
import {Observable, Subject, Subscription} from 'rxjs';
import {takeUntil} from 'rxjs/operators';

import {InstallerStateService} from '../../../core/services/installer-state.service';
import {AppDeployAdapterConfig, AppDeployGatewayConfig, AppDeployImageConfig, AppDeployRegistryConfig, AppDeploySecurityConfig, DeploymentGoal, InstallerState} from '../../types/installer.types';

// Custom async validator for JWKS file content
const jwksJsonValidator: AsyncValidatorFn =
    (control: AbstractControl): Promise<ValidationErrors|null> => {
      const file = control.value;
      if (!file || !(file instanceof File)) {
        return Promise.resolve(null);  // Optional field, no file selected
      }

      return new Promise((resolve) => {
        const reader = new FileReader();
        reader.onload = () => {
          try {
            JSON.parse(reader.result as string);
            resolve(null);  // Valid JSON
          } catch (e) {
            console.error('JWKS file content is not valid JSON:', e);
            resolve({invalidJson: true});
          }
        };
        reader.onerror = () => {
          console.error('Error reading JWKS file');
          resolve({fileReadError: true});
        };
        reader.readAsText(file);
      });
    };

@Component({
  selector: 'app-step-app-config',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatRadioModule,
    MatCheckboxModule,
    MatTabsModule,
    MatProgressSpinnerModule,
    MatCardModule,
    MatSlideToggleModule,
    MatTooltipModule,
    ClipboardModule,
    MatSnackBarModule,
  ],
  templateUrl: './step-app-config.component.html',
  styleUrls: ['./step-app-config.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepAppConfigComponent implements OnInit, OnDestroy {
  @Input() currentWizardStep: number = 0;
  @Output() goBackToPreviousWizardStep = new EventEmitter<void>();
  @ViewChild('componentConfigTabs') componentConfigTabs!: MatTabGroup;
  @ViewChild('adapterSubTabs') adapterSubTabs!: MatTabGroup;

  imageConfigForm!: FormGroup;
  registryConfigForm!: FormGroup;
  gatewayConfigForm!: FormGroup;
  adapterConfigForm!: FormGroup;
  securityConfigForm!: FormGroup;

  showGatewayTab: boolean = false;
  showAdapterTab: boolean = false;
  isAppDeploying: boolean = false;
  selectedJwkFileName?: string|' ';


  installerState!: InstallerState;
  private unsubscribe$ = new Subject<void>();

  private readonly URL_REGEX = /^(https?|ftp):\/\/[^\s/$.?#].[^\s]*$/i;

  currentInternalStep: number = 0;
  totalInternalSteps: number = 0;

  constructor(
      private fb: FormBuilder,
      private installerStateService: InstallerStateService,
      protected cdr: ChangeDetectorRef,
      private clipboard: Clipboard,
      private router: Router,
      private snackBar: MatSnackBar,
  ) {}

  ngOnInit(): void {
    this.initializeForms();
    this.installerStateService.installerState$
        .pipe(takeUntil(this.unsubscribe$))
        .subscribe(state => {
          this.installerState = state;
          this.updateTabVisibility(state.deploymentGoal);
          this.patchFormValuesFromState(state);
          this.setConditionalImageFormValidators(state.deploymentGoal);
          this.updateTotalInternalSteps();
          this.cdr.detectChanges();
        });
    this.adapterConfigForm.get('enableSchemaValidation')?.valueChanges
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe(value => {
         console.log('DEBUG: adapterConfigForm.enableSchemaValidation valueChanges:', value);
        this.cdr.detectChanges();
      });

    this.securityConfigForm.get('enableInBoundAuth')
        ?.valueChanges.pipe(takeUntil(this.unsubscribe$))
        .subscribe(enabled => {
          const issuerUrlCtrl = this.securityConfigForm.get('issuerUrl');
          const jwksFileCtrl = this.securityConfigForm.get('jwksFile');
          const idClaimCtrl = this.securityConfigForm.get('idclaim');
          const allowedValuesCtrl =
              this.securityConfigForm.get('allowedValues');

          if (enabled) {
            issuerUrlCtrl?.setValidators([Validators.required]);
            idClaimCtrl?.setValidators([Validators.required]);
            allowedValuesCtrl?.setValidators([Validators.required]);
            // jwksFile is optional
            jwksFileCtrl?.clearValidators();
          } else {
            issuerUrlCtrl?.clearValidators();
            jwksFileCtrl?.clearValidators();
            idClaimCtrl?.clearValidators();
            allowedValuesCtrl?.clearValidators();
          }
          issuerUrlCtrl?.updateValueAndValidity();
          jwksFileCtrl?.updateValueAndValidity();
          idClaimCtrl?.updateValueAndValidity();
          allowedValuesCtrl?.updateValueAndValidity();
        });

    this.securityConfigForm.get('enableOutBoundAuth')
        ?.valueChanges.pipe(takeUntil(this.unsubscribe$))
        .subscribe(enabled => {
          const audOverridesCtrl = this.securityConfigForm.get('audOverrides');
          audOverridesCtrl?.clearValidators();
          audOverridesCtrl?.updateValueAndValidity();
        });
  }

  ngOnDestroy(): void {
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

  private initializeForms(): void {
    this.imageConfigForm = this.fb.group({
      registryImageUrl: [''],
      registryAdminImageUrl: [''],
      gatewayImageUrl: [''],
      adapterImageUrl: [''],
      subscriptionImageUrl: [''],
    });
    this.registryConfigForm = this.fb.group({
      registryUrl: ['', [Validators.required, Validators.pattern(this.URL_REGEX)]],
      registryKeyId: ['', Validators.required],
      registrySubscriberId: ['', Validators.required],
      enableAutoApprover: [false]
    });
    this.gatewayConfigForm = this.fb.group({
      gatewaySubscriptionId: ['', Validators.required],
    });
    this.adapterConfigForm = this.fb.group({
      enableSchemaValidation: [false],
    });
    this.securityConfigForm = this.fb.group({
      enableInBoundAuth: [false],
      enableOutBoundAuth: [false],
      issuerUrl: [''],
      idclaim: [''],
      allowedValues: [''],
      jwksFile: ['', null, jwksJsonValidator],  // Apply the async validator
      audOverrides: [''],
    });
  }

  onJwkFileSelected(event: any): void {
    const file: File = event.target.files[0];

    if (file) {
      this.selectedJwkFileName = file.name;

      // Update your form control. Assuming you kept the name 'jwksUrl' or
      // renamed it to 'jwkFile'
      this.securityConfigForm.patchValue({jwksFile: file}, {emitEvent: true});

      // Mark as touched so validators trigger and errors are shown.
      this.securityConfigForm.get('jwksFile')?.markAsTouched();
    } else {
      // If file selection is cancelled or no file is selected, clear the
      // control.
      this.selectedJwkFileName = undefined;
      this.securityConfigForm.patchValue({jwksFile: ''}, {emitEvent: true});
      this.securityConfigForm.get('jwksFile')?.updateValueAndValidity();
    }
  }

  private updateTabVisibility(goal: DeploymentGoal): void {
    this.showGatewayTab = goal.all || goal.gateway;
    this.showAdapterTab = goal.all ||
      goal.bap || goal.bpp;
  }

  private setConditionalImageFormValidators(goal: DeploymentGoal): void {
    const { all, registry, gateway, bap, bpp } = goal;
    const controls = this.imageConfigForm.controls;

    if (all || registry) {
      controls['registryImageUrl'].setValidators(Validators.required);
      controls['registryAdminImageUrl'].setValidators(Validators.required);
    } else {
      controls['registryImageUrl'].clearValidators();
      controls['registryAdminImageUrl'].clearValidators();
      controls['registryImageUrl'].setValue('');
      controls['registryAdminImageUrl'].setValue('');
    }

    if (all || gateway) {
      controls['gatewayImageUrl'].setValidators(Validators.required);
    } else {
      controls['gatewayImageUrl'].clearValidators();
      controls['gatewayImageUrl'].setValue('');
    }

    if (all || bap || bpp) {
      controls['adapterImageUrl'].setValidators(Validators.required);
    } else {
      controls['adapterImageUrl'].clearValidators();
      controls['adapterImageUrl'].setValue('');
    }

    if (all || gateway || bap || bpp) {
      controls['subscriptionImageUrl'].setValidators(Validators.required);
    } else {
      controls['subscriptionImageUrl'].clearValidators();
      controls['subscriptionImageUrl'].setValue('');
    }

    Object.values(controls).forEach(control => control.updateValueAndValidity());
    this.imageConfigForm.updateValueAndValidity();
    this.cdr.detectChanges();
  }

  private patchFormValuesFromState(state: InstallerState): void {
    if (state.appDeployImageConfig) {
      this.imageConfigForm.patchValue(state.appDeployImageConfig, { emitEvent: false });
    } else {
      const imagePatchObject: { [key: string]: string |
        undefined } = {};
      if (state.deploymentGoal.all || state.deploymentGoal.registry) {
        imagePatchObject['registryImageUrl'] = state.dockerImageConfigs?.find(c => c.component === 'registry')?.imageUrl;
        imagePatchObject['registryAdminImageUrl'] = state.dockerImageConfigs?.find(c => c.component === 'registry_admin')?.imageUrl;
      }
      if (this.showGatewayTab || this.showAdapterTab) {
        imagePatchObject['subscriptionImageUrl'] = state.dockerImageConfigs?.find(c => c.component === 'subscriber')?.imageUrl;
      }
      if (this.showGatewayTab) {
        imagePatchObject['gatewayImageUrl'] = state.dockerImageConfigs?.find(c => c.component === 'gateway')?.imageUrl;
      }
      if (this.showAdapterTab) {
        imagePatchObject['adapterImageUrl'] = state.dockerImageConfigs?.find(c => c.component === 'adapter')?.imageUrl;
      }
      this.imageConfigForm.patchValue(imagePatchObject, { emitEvent: false });
    }

    if (state.appDeployRegistryConfig) {
      this.registryConfigForm.patchValue(state.appDeployRegistryConfig, { emitEvent: false });
    } else {
      const registryAppConfig = state.appSpecificConfigs?.find(c => c.component === 'registry')?.configs;
      const registryUrlFromInfra = state.infraDetails?.registry_url?.value;
      this.registryConfigForm.patchValue({
        registryUrl: (registryAppConfig && registryAppConfig['registry_url']) ? registryAppConfig['registry_url'] : registryUrlFromInfra || '',
        registryKeyId: registryAppConfig?.['key_id'] || '',
        registrySubscriberId: registryAppConfig?.['subscriber_id'] || '',
        enableAutoApprover: registryAppConfig?.['enable_auto_approver'] ?? false,
      }, { emitEvent: false });
    }

    if (state.appDeployGatewayConfig) {
      this.gatewayConfigForm.patchValue(state.appDeployGatewayConfig, { emitEvent: false });
    } else {
      const gatewayAppConfig = state.appSpecificConfigs?.find(c => c.component === 'gateway')?.configs;
      if (gatewayAppConfig) {
        this.gatewayConfigForm.patchValue({
          gatewaySubscriptionId: gatewayAppConfig['subscriber_id'] || '',
        }, { emitEvent: false });
      }
    }

    if (state.appDeployAdapterConfig) {
      this.adapterConfigForm.patchValue(state.appDeployAdapterConfig, { emitEvent: false });
    } else {
      const adapterAppConfig = state.appSpecificConfigs?.find(c => c.component === 'adapter')?.configs;
      if (adapterAppConfig) {
        const enableSchemaValidation = adapterAppConfig['enable_schema_validation'] || false;
        this.adapterConfigForm.patchValue({
          enableSchemaValidation: enableSchemaValidation,
        }, { emitEvent: false });
      }
    }

    if (state.appDeploySecurityConfig) {
      this.securityConfigForm.patchValue(
          state.appDeploySecurityConfig, {emitEvent: false});
    }
  }

  get isAppConfigValid(): boolean {
    console.log('--- Checking isAppConfigValid ---');

    if (this.imageConfigForm.invalid) {
      console.log('isAppConfigValid: false (imageConfigForm is invalid)');
      console.log('Image Form Errors:', this.imageConfigForm.errors);
      console.log('Image Form Controls status:', this.imageConfigForm.controls);
      return false;
    }
    if (this.registryConfigForm.invalid) {
      console.log('isAppConfigValid: false (registryConfigForm is invalid)');
      console.log('Registry Form Errors:', this.registryConfigForm.errors);
      console.log('Registry Form Controls status:', this.registryConfigForm.controls);
      return false;
    }

    const goal = this.installerState.deploymentGoal;
    console.log('Deployment Goal:', goal);

    if ((goal.all || goal.gateway)) {
      console.log('Gateway deployment enabled. Checking gatewayConfigForm...');
      if (this.gatewayConfigForm.invalid) {
        console.log('isAppConfigValid: false (gatewayConfigForm is invalid)');
        console.log('Gateway Form Errors:', this.gatewayConfigForm.errors);
        console.log('Gateway Form Controls status:', this.gatewayConfigForm.controls);
        return false;
      } else {
        console.log('gatewayConfigForm is valid.');
      }
    } else {
      console.log('Gateway deployment not enabled. Skipping gatewayConfigForm check.');
    }

    if (goal.all || goal.bap || goal.bpp) {
      console.log('Adapter deployment enabled. Checking adapterConfigForm...');
      // Mark adapter form as touched here to show errors immediately
      this.adapterConfigForm.markAllAsTouched();
      if (this.adapterConfigForm.invalid) {
        console.log('isAppConfigValid: false (adapterConfigForm is invalid)');
        console.log('Adapter Form Errors:', this.adapterConfigForm.errors);
        console.log('Adapter Form Controls status:', this.adapterConfigForm.controls);
        return false;
      } else {
        console.log('adapterConfigForm is valid.');
      }

    } else {
      console.log('Adapter/BAP/BPP deployment not enabled. Skipping adapterConfigForm and file checks.');
    }

    if (this.securityConfigForm.invalid) {
      console.log('isAppConfigValid: false (securityConfigForm is invalid)');
      console.log('Security Form Errors:', this.securityConfigForm.errors);
      return false;
    }

    console.log('--- isAppConfigValid: TRUE ---');
    return true;
  }

  getErrorMessage(control: AbstractControl | null, fieldName: string): string {
    if (!control || (!control.touched && !control.dirty)) {
      return '';
    }
    if (control.hasError('required')) {
      return `${fieldName} is required.`;
    }
    if (control.hasError('pattern')) {
      return `Please enter a valid ${fieldName}.`;
    }
    if (control.hasError('invalidJson')) {
      return `${fieldName} must be a valid JSON file.`;
    }
    if (control.hasError('fileReadError')) {
      return `Error reading the ${fieldName} file.`;
    }
    return '';
  }

  private updateTotalInternalSteps(): void {
    let count = 2;
    // Image Config (0) + Registry Config (1) are always visible

    if (this.showGatewayTab) {
      count++;
    }
    if (this.showAdapterTab) {
      count++;
    }
    // Security Config
    count++;
    this.totalInternalSteps = count;
    console.log('totalInternalSteps:', this.totalInternalSteps);
  }

  public isLastConfigTabActive(): boolean {
    if (!this.componentConfigTabs) {
      console.log('isLastConfigTabActive: componentConfigTabs not ready.');
      return false;
    }
    const currentSelectedMainTabIndex = this.componentConfigTabs.selectedIndex;
    const lastExpectedTabIndex = this.totalInternalSteps - 1;
    const isLast = currentSelectedMainTabIndex === lastExpectedTabIndex;
    console.log(`isLastConfigTabActive: current main tab index = ${currentSelectedMainTabIndex}, expected last tab index = ${lastExpectedTabIndex}, is last = ${isLast}`);
    return isLast;
  }

  public isCurrentMainTabValid(): boolean {
    if (!this.componentConfigTabs) {
      return false;
    }

    const currentTabIndex = this.componentConfigTabs.selectedIndex;

    const visibleTabs = [
      { index: 0, form: this.imageConfigForm, name: 'Image Config' },
      { index: 1, form: this.registryConfigForm, name: 'Registry Config' },
    ];
    if (this.showGatewayTab) {
      visibleTabs.push({ index: visibleTabs.length, form: this.gatewayConfigForm, name: 'Gateway Config' });
    }
    if (this.showAdapterTab) {
      visibleTabs.push({ index: visibleTabs.length, form: this.adapterConfigForm, name: 'Adapter Config' });
    }

    visibleTabs.push({
      index: visibleTabs.length,
      form: this.securityConfigForm,
      name: 'Security Config'
    });

    const currentVisibleTab = visibleTabs.find(tab => tab.index === currentTabIndex);
    if (currentVisibleTab) {
      return currentVisibleTab.form.valid;
    }
    return false;
  }

  public onNextTab(): void {
    if (this.componentConfigTabs) {
      const currentTabIndex = this.componentConfigTabs.selectedIndex;
      if (typeof currentTabIndex === 'number') {
        this.saveCurrentTabConfigToState(currentTabIndex);
        if (currentTabIndex < (this.totalInternalSteps - 1)) {
          this.componentConfigTabs.selectedIndex = currentTabIndex + 1;
          this.currentInternalStep = this.componentConfigTabs.selectedIndex;
          this.cdr.detectChanges();
        }
      }
    }
  }

  public onPreviousTab(): void {
    if (this.componentConfigTabs) {
      const currentTabIndex = this.componentConfigTabs.selectedIndex;
      if (typeof currentTabIndex === 'number') {
        this.saveCurrentTabConfigToState(currentTabIndex);
        if (currentTabIndex > 0) {
          this.componentConfigTabs.selectedIndex = currentTabIndex - 1;
          this.currentInternalStep = this.componentConfigTabs.selectedIndex;
          this.cdr.detectChanges();
        } else {
          this.router.navigate(['installer', 'domain-configuration']);
          console.log('Emitting goBackToPreviousWizardStep event...');
        }
      }
    }
  }

  public onNextSubTab(currentIndex: number): void {
    if (this.adapterSubTabs) {
      if (currentIndex < (this.adapterSubTabs._tabs?.length ?? 0) - 1) {
        this.adapterSubTabs.selectedIndex = currentIndex + 1;
        this.cdr.detectChanges();
      } else {
        this.onNextTab();
      }
    }
  }

  public onPreviousSubTab(currentIndex: number): void {
    if (this.adapterSubTabs) {
      if (currentIndex > 0) {
        this.adapterSubTabs.selectedIndex = currentIndex - 1;
        this.cdr.detectChanges();
      } else {
        let adapterTabIndex = 2;
        if (this.showGatewayTab) {
          adapterTabIndex = 3;
        }

        if (this.componentConfigTabs) {
          this.componentConfigTabs.selectedIndex = adapterTabIndex - 1;
          this.currentInternalStep = this.componentConfigTabs.selectedIndex;
          this.cdr.detectChanges();
        }
      }
    }
  }

  private saveCurrentTabConfigToState(currentTabIndex: number): void {
    let formToSave: FormGroup |
      null = null;
    let formName: string = '';

    // Determine which form is active based on the current tab index
    // Assuming the order of tabs is consistent: Image (0), Registry (1), Gateway (if visible), Adapter (if visible)
    if (currentTabIndex === 0) {
      formToSave = this.imageConfigForm;
      formName = 'Image Config';
    } else if (currentTabIndex === 1) {
      formToSave = this.registryConfigForm;
      formName = 'Registry Config';
    } else if (this.showGatewayTab && currentTabIndex === 2) {
      formToSave = this.gatewayConfigForm;
      formName = 'Gateway Config';
    } else if (this.showAdapterTab && currentTabIndex === (this.showGatewayTab ? 3 : 2)) {
      formToSave = this.adapterConfigForm;
      formName = 'Adapter Config';
    } else if (
        currentTabIndex ===
        (this.showAdapterTab ? (this.showGatewayTab ? 4 : 3) :
                               (this.showGatewayTab ? 3 : 2))) {
      formToSave = this.securityConfigForm;
      formName = 'Security Config';
    }

    if (formToSave) {
      formToSave.markAllAsTouched();
      if (formToSave.valid) {
        if (formToSave === this.imageConfigForm) {
          this.installerStateService.updateAppDeployImageConfig(formToSave.getRawValue());
        } else if (formToSave === this.registryConfigForm) {
          this.installerStateService.updateAppDeployRegistryConfig(formToSave.getRawValue());
        } else if (formToSave === this.gatewayConfigForm) {
          this.installerStateService.updateAppDeployGatewayConfig(formToSave.getRawValue());
        } else if (formToSave === this.adapterConfigForm) {
          this.installerStateService.updateAppDeployAdapterConfig(formToSave.getRawValue());
        } else if (formToSave === this.securityConfigForm) {
          this.installerStateService.updateAppDeploySecurityConfig(
              formToSave.getRawValue());
        }
        console.log(`Saved ${formName} config to state.`);
      } else {
        console.warn(`Form ${formName} is invalid. Not saving to state.`);
      }
    }
  }

  private readFileContent(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        resolve(reader.result as string);
      };
      reader.onerror = reject;
      reader.readAsText(file);
    });
  }

  public async proceedToConfigGeneration(): Promise<void> {
    this.saveCurrentTabConfigToState(this.currentInternalStep);

    this.imageConfigForm.markAllAsTouched();
    this.registryConfigForm.markAllAsTouched();
    if (this.showGatewayTab) this.gatewayConfigForm.markAllAsTouched();
    if (this.showAdapterTab) this.adapterConfigForm.markAllAsTouched();
    this.securityConfigForm.markAllAsTouched();

    if (!this.isAppConfigValid) {
      console.warn(
          'Cannot proceed: One or more configuration forms are invalid.');
      this.cdr.detectChanges();
      return;
    }

    const securityConfigRaw = this.securityConfigForm.getRawValue();
    let jwksContent = '';

    // Process the JWKS file if inbound auth is enabled
    if (securityConfigRaw.enableInBoundAuth) {
      const jwksFileControl = this.securityConfigForm.get('jwksFile');
      if (jwksFileControl && !jwksFileControl.errors &&
          securityConfigRaw.jwksFile instanceof File) {
        try {
          const rawContent =
              await this.readFileContent(securityConfigRaw.jwksFile);
          const parsedJson = JSON.parse(rawContent);
          const compactJsonString = JSON.stringify(parsedJson);
          jwksContent =
              compactJsonString.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
        } catch (e) {
          console.error('Unexpected error parsing JWKS file:', e);
          this.snackBar.open(
              'Failed to parse JWKS file. Please ensure it is a valid JSON file.',
              'Close', {
                duration: 5000,
                panelClass: ['error-snackbar'],
              });
          this.cdr.detectChanges();
          return;  // Stop navigation if file parsing fails
        }
      }
    }

    // Update State
    this.installerStateService.updateAppDeployImageConfig(
        this.imageConfigForm.getRawValue());
    this.installerStateService.updateAppDeployRegistryConfig(
        this.registryConfigForm.getRawValue());
    if (this.showGatewayTab)
      this.installerStateService.updateAppDeployGatewayConfig(
          this.gatewayConfigForm.getRawValue());
    if (this.showAdapterTab)
      this.installerStateService.updateAppDeployAdapterConfig(
          this.adapterConfigForm.getRawValue());

    // Replace the raw File object with the processed string before saving to
    // state
    const finalSecurityConfigToSave: AppDeploySecurityConfig = {
      enableInBoundAuth: securityConfigRaw.enableInBoundAuth,
      enableOutBoundAuth: securityConfigRaw.enableOutBoundAuth,
      issuerUrl: securityConfigRaw.issuerUrl,
      idclaim: securityConfigRaw.idclaim,
      allowedValues: securityConfigRaw.allowedValues,
      audOverrides: securityConfigRaw.audOverrides,
      jwksFileContent:
          jwksContent,  // Storing the parsed string instead of the File
    };
    this.installerStateService.updateAppDeploySecurityConfig(
        finalSecurityConfigToSave);

    this.installerStateService.updateState({isAppConfigValid: true});

    // Navigate to the next step
    void this.router.navigate(['installer', 'view-config']);
  }
}