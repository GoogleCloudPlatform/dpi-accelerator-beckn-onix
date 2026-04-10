/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
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

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {removeEmptyValues, sanitizeFormValues} from '../../../shared/utils';
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

  @ViewChild('jwkFileInput') jwkFileInput!: ElementRef<HTMLInputElement>;

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

  private readonly URL_REGEX = /^\s*(https?|ftp):\/\/[^\s/$.?#].[^\s]*\s*$/i;

  currentInternalStep: number = 0;
  totalInternalSteps: number = 0;

  isGeneratingConfigs: boolean = false;
  configGenerationError: string|null = null;

  constructor(
      private fb: FormBuilder,
      private installerStateService: InstallerStateService,
      protected cdr: ChangeDetectorRef, private clipboard: Clipboard,
      private router: Router, private snackBar: MatSnackBar,
      private apiService: ApiService) {}

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
      jwksFile: ['', null, jwksJsonValidator],
      audOverrides: [''],
    });
  }

  onJwkFileSelected(event: any): void {
    const file: File = event.target.files[0];

    if (file) {
      this.selectedJwkFileName = file.name;
      this.securityConfigForm.patchValue({jwksFile: file}, {emitEvent: true});
      this.securityConfigForm.get('jwksFile')?.markAsTouched();
    } else {
      this.selectedJwkFileName = undefined;
      this.securityConfigForm.patchValue({jwksFile: ''}, {emitEvent: true});
      this.securityConfigForm.get('jwksFile')?.updateValueAndValidity();
    }
  }

  clearJwkFile(event: Event): void {
    event.stopPropagation();

    this.selectedJwkFileName = undefined;
    this.securityConfigForm.patchValue({jwksFile: ''}, {emitEvent: true});
    this.securityConfigForm.get('jwksFile')?.markAsUntouched();
    this.securityConfigForm.get('jwksFile')?.setErrors(null);
    this.securityConfigForm.get('jwksFile')?.updateValueAndValidity();

    if (this.jwkFileInput && this.jwkFileInput.nativeElement) {
      this.jwkFileInput.nativeElement.value = '';
    }

    // Immediately wipe the file content from the global state
    const currentState = this.installerStateService.getCurrentState();
    if (currentState.appDeploySecurityConfig) {
      this.installerStateService.updateAppDeploySecurityConfig({
        ...currentState.appDeploySecurityConfig,
        jwksFileContent: '',
        jwksFileName: ''
      });
    }
  }

  private updateTabVisibility(goal: DeploymentGoal): void {
    this.showGatewayTab = goal.all || goal.gateway;
    this.showAdapterTab = goal.all || goal.bap || goal.bpp;
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
      const imagePatchObject: {[key: string]: string|undefined} = {};
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

      // Restore the File object so the UI and validators know it exists
      if (state.appDeploySecurityConfig.jwksFileContent) {
        this.selectedJwkFileName =
            state.appDeploySecurityConfig.jwksFileName || 'uploaded-jwks.json';

        const restoredFile = new File(
            [state.appDeploySecurityConfig.jwksFileContent],
            this.selectedJwkFileName, {type: 'application/json'});

        this.securityConfigForm.patchValue(
            {jwksFile: restoredFile}, {emitEvent: false});
      }
    }
  }

  get isAppConfigValid(): boolean {
    if (this.imageConfigForm.invalid) {
      return false;
    }
    if (this.registryConfigForm.invalid) {
      return false;
    }

    const goal = this.installerState.deploymentGoal;

    if ((goal.all || goal.gateway)) {
      if (this.gatewayConfigForm.invalid) {
        return false;
      }
    }

    if (goal.all || goal.bap || goal.bpp) {
      this.adapterConfigForm.markAllAsTouched();
      if (this.adapterConfigForm.invalid) {
        return false;
      }
    }

    if (this.securityConfigForm.invalid) {
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
    let count = 2;  // Image Config (0) + Registry Config (1) are always visible
    if (this.showGatewayTab) count++;
    if (this.showAdapterTab) count++;
    count++;  // Security Config
    this.totalInternalSteps = count;
  }

  public isLastConfigTabActive(): boolean {
    if (!this.componentConfigTabs) {
      return false;
    }
    const currentSelectedMainTabIndex = this.componentConfigTabs.selectedIndex;
    const lastExpectedTabIndex = this.totalInternalSteps - 1;
    return currentSelectedMainTabIndex === lastExpectedTabIndex;
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
        if (this.showGatewayTab) adapterTabIndex = 3;

        if (this.componentConfigTabs) {
          this.componentConfigTabs.selectedIndex = adapterTabIndex - 1;
          this.currentInternalStep = this.componentConfigTabs.selectedIndex;
          this.cdr.detectChanges();
        }
      }
    }
  }

  private saveCurrentTabConfigToState(currentTabIndex: number): void {
    let formToSave: FormGroup|null = null;
    let formName: string = '';

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
        const sanitizedValue = sanitizeFormValues(formToSave.getRawValue());
        if (formToSave === this.imageConfigForm) {
          this.installerStateService.updateAppDeployImageConfig(sanitizedValue);
        } else if (formToSave === this.registryConfigForm) {
          this.installerStateService.updateAppDeployRegistryConfig(
              sanitizedValue);
        } else if (formToSave === this.gatewayConfigForm) {
          this.installerStateService.updateAppDeployGatewayConfig(
              sanitizedValue);
        } else if (formToSave === this.adapterConfigForm) {
          this.installerStateService.updateAppDeployAdapterConfig(
              sanitizedValue);
        } else if (formToSave === this.securityConfigForm) {
          this.installerStateService.updateAppDeploySecurityConfig(
              sanitizedValue);
        }
      }
    }
  }

  private readFileContent(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
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

    this.installerStateService.updateAppDeployImageConfig(
        sanitizeFormValues(this.imageConfigForm.getRawValue()));
    this.installerStateService.updateAppDeployRegistryConfig(
        sanitizeFormValues(this.registryConfigForm.getRawValue()));
    if (this.showGatewayTab) {
      this.installerStateService.updateAppDeployGatewayConfig(
          sanitizeFormValues(this.gatewayConfigForm.getRawValue()));
    }
    if (this.showAdapterTab) {
      this.installerStateService.updateAppDeployAdapterConfig(
          sanitizeFormValues(this.adapterConfigForm.getRawValue()));
    }

    const securityConfigRaw =
        sanitizeFormValues(this.securityConfigForm.getRawValue());
    let jwksContent = '';

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
          return;
        }
      }
    }

    const finalSecurityConfigToSave: AppDeploySecurityConfig = {
      enableInBoundAuth: securityConfigRaw.enableInBoundAuth,
      enableOutBoundAuth: securityConfigRaw.enableOutBoundAuth,
      issuerUrl: securityConfigRaw.issuerUrl,
      idclaim: securityConfigRaw.idclaim,
      allowedValues: securityConfigRaw.allowedValues,
      audOverrides: securityConfigRaw.audOverrides,
      jwksFileContent: jwksContent,
      jwksFileName: this.selectedJwkFileName
    };

    this.installerStateService.updateAppDeploySecurityConfig(
        finalSecurityConfigToSave);

    // --- API CALL LOGIC STARTS HERE ---
    this.isGeneratingConfigs = true;
    this.configGenerationError = null;
    this.cdr.detectChanges();

    const payloadState = this.installerStateService.getCurrentState();

    const potentialDomainNames = {
      registry: payloadState.subdomainConfigs
                    ?.find((c: any) => c.component === 'registry')
                    ?.subdomainName,
      registry_admin: payloadState.subdomainConfigs
                          ?.find((c: any) => c.component === 'registry_admin')
                          ?.subdomainName,
      subscriber: payloadState.subdomainConfigs
                      ?.find((c: any) => c.component === 'subscriber')
                      ?.subdomainName,
      gateway: payloadState.subdomainConfigs
                   ?.find((c: any) => c.component === 'gateway')
                   ?.subdomainName,
      adapter: payloadState.subdomainConfigs
                   ?.find((c: any) => c.component === 'adapter')
                   ?.subdomainName
    };

    const isDeployingGateway = payloadState.deploymentGoal.gateway ||
        payloadState.deploymentGoal.all || false;
    const isDeployingAdapter = payloadState.deploymentGoal.bap ||
        payloadState.deploymentGoal.bpp || payloadState.deploymentGoal.all ||
        false;
    const isDeployingRegistry = payloadState.deploymentGoal.registry ||
        payloadState.deploymentGoal.all || false;

    const fullPayload: any = {
      'app_name': payloadState.appName,
      'domain_names': removeEmptyValues(potentialDomainNames),
      'components': {
        'bap': payloadState.deploymentGoal.bap || false,
        'bpp': payloadState.deploymentGoal.bpp || false,
        'registry': isDeployingRegistry,
        'gateway': isDeployingGateway,
        'adapter': isDeployingAdapter
      },
      'registry_url': payloadState.appDeployRegistryConfig?.registryUrl,
      'registry_config': {
        'subscriber_id':
            payloadState.appDeployRegistryConfig?.registrySubscriberId || '',
        'key_id': payloadState.appDeployRegistryConfig?.registryKeyId || '',
        'enable_auto_approver':
            payloadState.appDeployRegistryConfig?.enableAutoApprover || false
      }
    };

    if (isDeployingGateway) {
      fullPayload.gateway_config = {
        'subscriber_id':
            payloadState.appDeployGatewayConfig?.gatewaySubscriptionId || ''
      };
    }

    if (isDeployingAdapter) {
      fullPayload.adapter_config = {
        'enable_schema_validation':
            payloadState.appDeployAdapterConfig?.enableSchemaValidation || false
      };
    }

    if (finalSecurityConfigToSave.enableInBoundAuth ||
        finalSecurityConfigToSave.enableOutBoundAuth) {
      fullPayload.security_config = {
        'enable_inbound_auth':
            finalSecurityConfigToSave.enableInBoundAuth || false,
        'issuer_url': finalSecurityConfigToSave.enableInBoundAuth ?
            (finalSecurityConfigToSave.issuerUrl || '') :
            '',
        'jwks_content': finalSecurityConfigToSave.enableInBoundAuth ?
            (finalSecurityConfigToSave.jwksFileContent || '') :
            '',
        'enable_outbound_auth':
            finalSecurityConfigToSave.enableOutBoundAuth || false,
        'aud_overrides': finalSecurityConfigToSave.enableOutBoundAuth ?
            (finalSecurityConfigToSave.audOverrides || '') :
            '',
        'idclaim': finalSecurityConfigToSave.enableInBoundAuth ?
            (finalSecurityConfigToSave.idclaim || '') :
            '',
        'allowed_values': finalSecurityConfigToSave.enableInBoundAuth ?
            (finalSecurityConfigToSave.allowedValues ?
                 finalSecurityConfigToSave.allowedValues.split(',').map(
                     s => s.trim()) :
                 []) :
            []
      };
    }

    this.apiService.postConfigs(fullPayload).subscribe({
      next: () => {
        this.isGeneratingConfigs = false;
        this.installerStateService.updateState({isAppConfigValid: true});
        this.router.navigate(['installer', 'view-config']);
      },
      error: (err) => {
        console.error('Config generation failed:', err);
        this.isGeneratingConfigs = false;

        let errorMsg =
            'Failed to generate configurations. Please check your inputs and try again.';
        if (err.status === 422 && err.error && err.error.detail) {
          if (Array.isArray(err.error.detail)) {
            const missingFields =
                err.error.detail
                    .map((d: any) => `${d.loc.join('.')} (${d.msg})`)
                    .join(', ');
            errorMsg = `Backend Validation Error: ${missingFields}`;
          } else {
            errorMsg = `Validation Error: ${JSON.stringify(err.error.detail)}`;
          }
        } else if (err.error && err.error.message) {
          errorMsg = err.error.message;
        }

        this.configGenerationError = errorMsg;

        // Show fallback toast notification
        this.snackBar.open(this.configGenerationError, 'Close', {
          duration: 7000,
          panelClass: ['error-snackbar'],
        });

        this.cdr.markForCheck();
        this.cdr.detectChanges();
      }
    });
  }
}