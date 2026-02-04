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
import {AbstractControl, FormBuilder, FormControl, FormGroup, ReactiveFormsModule, ValidationErrors, Validators} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatCheckboxModule} from '@angular/material/checkbox';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatRadioModule} from '@angular/material/radio';
import {MatTabGroup, MatTabsModule} from '@angular/material/tabs';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Router} from '@angular/router';
import {EMPTY, Subject, Subscription} from 'rxjs';
import {catchError, finalize, takeUntil} from 'rxjs/operators';
import {trySanitizeUrl} from 'safevalues';
import {windowOpen} from 'safevalues/dom';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {removeEmptyValues} from '../../../shared/utils';
import {AppDeployAdapterConfig, AppDeployGatewayConfig, AppDeployImageConfig, AppDeployRegistryConfig, BackendAppDeploymentRequest, DeploymentGoal, DomainConfig, InstallerState, SubdomainConfig} from '../../types/installer.types';

@Component({
  selector: 'app-step-app-deploy',
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
    MatTooltipModule,
    ClipboardModule,
  ],
  templateUrl: './step-deploy-app.component.html',
  styleUrls: ['./step-deploy-app.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepAppDeployComponent implements OnInit, OnDestroy {
  @Input() currentWizardStep: number = 0;
  @Output() deploymentInitiated = new EventEmitter<void>();
  @Output() deploymentComplete = new EventEmitter<void>();
  @Output() deploymentError = new EventEmitter<string>();
  @Output() goBackToPreviousWizardStep = new EventEmitter<void>();
  @ViewChild('componentConfigTabs') componentConfigTabs!: MatTabGroup;
  @ViewChild('adapterSubTabs') adapterSubTabs!: MatTabGroup;
  @ViewChild('appLogContainer') private appLogContainer!: ElementRef;

  imageConfigForm!: FormGroup;
  registryConfigForm!: FormGroup;
  gatewayConfigForm!: FormGroup;
  adapterConfigForm!: FormGroup;
  appDeploymentLogs: string[] = [];
  appExternalIp: string | null = null;

  serviceUrls: { [key: string]: string } = {};
  servicesDeployed: string[] = [];
  logsExplorerUrls: { [key: string]: string } = {};
  adapterLogButtonShown: boolean = false;

  showGatewayTab: boolean = false;
  showAdapterTab: boolean = false;
  showAdapterLogButton: boolean = false;

  installerState!: InstallerState;
  private appWsSubscription!: Subscription;
  private unsubscribe$ = new Subject<void>();

  private readonly URL_REGEX = /^(https?|ftp):\/\/[^\s/$.?#].[^\s]*$/i;

  currentInternalStep: number = 0;
  totalInternalSteps: number = 0;

  constructor(
    private fb: FormBuilder,
    private installerStateService: InstallerStateService,
    protected cdr: ChangeDetectorRef,
    private webSocketService: WebSocketService,
    private clipboard: Clipboard,
    private router: Router
  ) { }

  ngOnInit(): void {
    this.initializeForms();
    this.installerStateService.installerState$
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe(state => {
        this.installerState = state;
      if (state.appDeploymentStatus === 'completed') {
      this.serviceUrls = state.deployedServiceUrls || {};
      this.servicesDeployed = state.servicesDeployed || [];
      this.logsExplorerUrls = state.logsExplorerUrls || {};
      this.appExternalIp = state.appExternalIp || null;
      this.showAdapterLogButton = Object.keys(this.serviceUrls).some(key => key.startsWith('adapter_'));
      }
        this.updateTabVisibility(state.deploymentGoal);
        this.patchFormValuesFromState(state);
        this.setConditionalImageFormValidators(state.deploymentGoal);
        this.updateTotalInternalSteps();
        this.cdr.detectChanges();
        if (this.isAppDeploying) {
          this.scrollToBottom();
        }
      });
    this.adapterConfigForm.get('enableSchemaValidation')?.valueChanges
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe(value => {
         console.log('DEBUG: adapterConfigForm.enableSchemaValidation valueChanges:', value);
        this.cdr.detectChanges();
      });
  }

  ngOnDestroy(): void {
    if (this.appWsSubscription) {
      this.appWsSubscription.unsubscribe();
    }
    this.webSocketService.closeConnection();
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
  }

  get isAppDeployStepValid(): boolean {
    console.log('--- Checking isAppDeployStepValid ---');
    if (this.isAppDeploying) {
      console.log('isAppDeployStepValid: false (isAppDeploying is true)');
      return false;
    }

    if (this.imageConfigForm.invalid) {
      console.log('isAppDeployStepValid: false (imageConfigForm is invalid)');
      console.log('Image Form Errors:', this.imageConfigForm.errors);
      console.log('Image Form Controls status:', this.imageConfigForm.controls);
      return false;
    }
    if (this.registryConfigForm.invalid) {
      console.log('isAppDeployStepValid: false (registryConfigForm is invalid)');
      console.log('Registry Form Errors:', this.registryConfigForm.errors);
      console.log('Registry Form Controls status:', this.registryConfigForm.controls);
      return false;
    }

    const goal = this.installerState.deploymentGoal;
    console.log('Deployment Goal:', goal);

    if ((goal.all || goal.gateway)) {
      console.log('Gateway deployment enabled. Checking gatewayConfigForm...');
      if (this.gatewayConfigForm.invalid) {
        console.log('isAppDeployStepValid: false (gatewayConfigForm is invalid)');
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
        console.log('isAppDeployStepValid: false (adapterConfigForm is invalid)');
        console.log('Adapter Form Errors:', this.adapterConfigForm.errors);
        console.log('Adapter Form Controls status:', this.adapterConfigForm.controls);
        return false;
      } else {
        console.log('adapterConfigForm is valid.');
      }

    } else {
      console.log('Adapter/BAP/BPP deployment not enabled. Skipping adapterConfigForm and file checks.');
    }

    console.log('--- isAppDeployStepValid: TRUE ---');
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
        }
        console.log(`Saved ${formName} config to state.`);
      } else {
        console.warn(`Form ${formName} is invalid. Not saving to state.`);
      }
    }
  }

  public async onDeployApp(): Promise<void> {
    this.saveCurrentTabConfigToState(this.currentInternalStep)
    this.imageConfigForm.markAllAsTouched();
    this.registryConfigForm.markAllAsTouched();
    if (this.showGatewayTab) this.gatewayConfigForm.markAllAsTouched();
    if (this.showAdapterTab) {
      this.adapterConfigForm.markAllAsTouched();
      const goal = this.installerState.deploymentGoal;
    }

    this.installerStateService.updateAppDeployImageConfig(this.imageConfigForm.getRawValue());
    this.installerStateService.updateAppDeployRegistryConfig(this.registryConfigForm.getRawValue());
    if (this.showGatewayTab) {
      this.installerStateService.updateAppDeployGatewayConfig(this.gatewayConfigForm.getRawValue());
    }
    if (this.showAdapterTab) {
      this.installerStateService.updateAppDeployAdapterConfig(this.adapterConfigForm.getRawValue());
    }


    if (!this.isAppDeployStepValid) {
      console.error('One or more application configuration forms are invalid. Please fill in all required fields to proceed.');
      this.cdr.detectChanges();
      return;
    }

    this.installerStateService.updateAppDeploymentStatus('in-progress');
    this.appDeploymentLogs = [];
    this.appExternalIp = null;
    this.serviceUrls = {};
    this.servicesDeployed = [];
    this.logsExplorerUrls = {};
    this.adapterLogButtonShown = false;
    this.cdr.detectChanges();

    this.deploymentInitiated.emit();

    const goal = this.installerState.deploymentGoal;

    const deployBap = goal.all ||
      goal.bap;
    const deployBpp = goal.all || goal.bpp;
    const deployRegistry = goal.all || goal.registry;
    const deployGateway = goal.all || goal.gateway;
    const deployAdapter = goal.all || goal.bap || goal.bpp;

    const imageConfigRaw = this.imageConfigForm.getRawValue();
    const registryConfigRaw = this.registryConfigForm.getRawValue();
    const gatewayConfigRaw = this.gatewayConfigForm.getRawValue();
    const adapterConfigRaw = this.adapterConfigForm.getRawValue();

    const subdomainConfigs: SubdomainConfig[] = this.installerState.subdomainConfigs || [];
    const globalDomainDetails: DomainConfig | null = this.installerState.globalDomainConfig;

    const potentialDomainNames = {
    registry: subdomainConfigs.find(c => c.component === 'registry')?.subdomainName,
    registry_admin: subdomainConfigs.find(c => c.component === 'registry_admin')?.subdomainName,
    subscriber: subdomainConfigs.find(c => c.component === 'subscriber')?.subdomainName,
    gateway: subdomainConfigs.find(c => c.component === 'gateway')?.subdomainName,
    adapter: subdomainConfigs.find(c => c.component === 'adapter')?.subdomainName
  };

   const potentialImageUrls = {
    registry: imageConfigRaw.registryImageUrl,
    registry_admin: imageConfigRaw.registryAdminImageUrl,
    subscriber: imageConfigRaw.subscriptionImageUrl,
    gateway: imageConfigRaw.gatewayImageUrl,
    adapter: imageConfigRaw.adapterImageUrl
  };

    const payload: BackendAppDeploymentRequest = {
      app_name: this.installerState.appName,
      components: {
        adapter: deployAdapter,
        gateway: deployGateway,
        registry: deployRegistry,
        bap: deployBap,
        bpp: deployBpp
      },

      domain_names: removeEmptyValues(potentialDomainNames),
      image_urls: removeEmptyValues(potentialImageUrls),

      registry_url: registryConfigRaw.registryUrl ||
        '',
      registry_config: {
        subscriber_id: registryConfigRaw.registrySubscriberId,
        key_id: registryConfigRaw.registryKeyId,
        enable_auto_approver: registryConfigRaw.enableAutoApprover
      },
      domain_config: {
        domainType: globalDomainDetails?.domainType ||
          'other_domain',
        baseDomain: globalDomainDetails?.baseDomain ||
          '',
        dnsZone: globalDomainDetails?.dnsZone ||
          ''
      }
    };
    if (deployGateway) {
      payload.gateway_config = {
        subscriber_id: gatewayConfigRaw.gatewaySubscriptionId ||
          ''
      };
    }

    if (deployAdapter) {
      payload.adapter_config = {
        enable_schema_validation: adapterConfigRaw.enableSchemaValidation,
      };
    }


    console.log('Final Application Deployment Payload:', payload);
    const wsUrl = `ws://localhost:8000/ws/deployApp`;
    this.appWsSubscription = this.webSocketService.connect(wsUrl)
    .pipe(
        takeUntil(this.unsubscribe$),
        catchError(error => {
            console.error('WebSocket connection error for app deployment:', error);
            const errorMessage = `WebSocket connection error: ${error.message || 'Could not connect to the backend server.'}`;

            this.installerStateService.updateAppDeploymentStatus('failed');

            this.appDeploymentLogs.push(errorMessage);
            this.deploymentError.emit(errorMessage);
            this.cdr.detectChanges();

            return EMPTY;
        }),
        finalize(() => {
            console.log('WebSocket stream for app deployment finalized.');
            // Check if the deployment was still "in-progress" when the socket closed.
            // This indicates an unexpected disconnection.
            if (this.installerState.appDeploymentStatus === 'in-progress') {
                const errorMessage = 'Deployment failed: The connection to the server was lost unexpectedly.';

                this.installerStateService.updateAppDeploymentStatus('failed');

                this.appDeploymentLogs.push(errorMessage);
                this.deploymentError.emit(errorMessage);
                this.cdr.detectChanges();
            }
            this.webSocketService.closeConnection();
        })
    )
    .subscribe({
        next: (message) => this.handleWebSocketMessage(message),
        error: (err) => {
            console.error('WebSocket runtime error during app deployment:', err);
            const errorMessage = `Deployment failed: ${err.message || 'An unknown error occurred during deployment.'}`;

            this.installerStateService.updateAppDeploymentStatus('failed');

            this.appDeploymentLogs.push(errorMessage);
            this.deploymentError.emit(errorMessage);
            this.cdr.detectChanges();
        },
        complete: () => {
            console.log('WebSocket connection for app deployment closed by the server.');
            this.cdr.detectChanges();
        }
    });
   this.webSocketService.sendMessage(payload);
  }

  private handleWebSocketMessage(message: any): void {
    let parsedMessage: any;
    try {
      parsedMessage = typeof message === 'string' ? JSON.parse(message) : message;
    } catch (e) {
      console.warn('Received non-JSON WebSocket message for app deployment:', message);
      this.appDeploymentLogs.push(String(message));
      this.cdr.detectChanges();
      this.scrollToBottom();
      return;
    }

    console.log('Received parsed message for app deployment:', parsedMessage);
    const { type, action, message: msgContent, data } = parsedMessage;
    switch (type) {
      case 'log':
        this.appDeploymentLogs.push(msgContent);
        break;
      case 'success':
       this.installerStateService.updateAppDeploymentStatus('completed');
        this.appDeploymentLogs.push('Application Deployment Completed Successfully!');
        if (data) {
        this.installerStateService.updateState({
        deployedServiceUrls: data.service_urls || {},
        servicesDeployed: data.services_deployed || [],
        logsExplorerUrls: data.logs_explorer_urls || {},
        appExternalIp: data.app_external_ip || null,
     });
      this.serviceUrls = data.service_urls || {};
      this.servicesDeployed = data.services_deployed || [];
      this.logsExplorerUrls = data.logs_explorer_urls || {};
      this.appExternalIp = data.app_external_ip || null;
      this.showAdapterLogButton = Object.keys(this.serviceUrls).some(key => key.startsWith('adapter_'));

        } else {
          console.warn('Success message data is missing or empty.');
        }

        this.deploymentComplete.emit();
        this.webSocketService.closeConnection();
        break;
      case 'error':
         this.installerStateService.updateAppDeploymentStatus('failed');
         const errorMessage = msgContent || 'Unknown deployment error.';
         this.appDeploymentLogs.push(`Application Deployment Failed: ${errorMessage}`);
         this.deploymentError.emit(errorMessage);
         this.webSocketService.closeConnection();
         break;
      default:
        this.appDeploymentLogs.push(`[UNKNOWN MESSAGE TYPE FOR APP DEPLOYMENT] ${JSON.stringify(parsedMessage)}`);
        break;
    }
    this.cdr.detectChanges();
    this.scrollToBottom();
  }

  private scrollToBottom(): void {
    try {
      if (this.appLogContainer && this.appLogContainer.nativeElement) {
        this.appLogContainer.nativeElement.scrollTop = this.appLogContainer.nativeElement.scrollHeight;
      }
    } catch (err) {
      console.error('Could not scroll app logs to bottom:', err);
    }
  }

  openUrl(url: string | null): void {
    if (url) {
        windowOpen(window, url, '_blank');
    }
  }

  setAdapterLogButtonShown(): boolean {
    this.adapterLogButtonShown = true;
    return true;
  }

  copyToClipboard(text: string | null): void {
    if (text) {
      this.clipboard.copy(text!);
      console.log('Copied to clipboard:', text);
    }
  }

  continueToNextStep(): void {
    this.router.navigate(['installer', 'health-checks']);
  }

  resetDeployment(): void {
    this.installerStateService.updateAppDeploymentStatus('pending');
    this.appDeploymentLogs = [];
    this.appExternalIp = null;
    this.serviceUrls = {};
    this.servicesDeployed = [];
    this.logsExplorerUrls = {};
    this.adapterLogButtonShown = false;
    this.cdr.detectChanges();
  }

  get isAppDeploying(): boolean {
  return this.installerState?.appDeploymentStatus === 'in-progress';
}

get appDeploymentComplete(): boolean {
  return this.installerState?.appDeploymentStatus === 'completed';
}

get appDeploymentFailed(): boolean {
  return this.installerState?.appDeploymentStatus === 'failed';
}
}