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

import {CommonModule} from '@angular/common';
import {ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, EventEmitter, OnDestroy, OnInit, Output, ViewChild} from '@angular/core';
import {FormBuilder, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatCheckbox} from '@angular/material/checkbox';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatSelectModule} from '@angular/material/select';
import {Router} from '@angular/router';
import {EMPTY, Subject} from 'rxjs';
import {catchError, debounceTime, distinctUntilChanged, finalize, takeUntil} from 'rxjs/operators';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {WebSocketService} from '../../../core/services/websocket.service';
import {sanitizeFormValues} from '../../../shared/utils';
import {appNameValidator} from '../../../shared/validators/custom-validators';
import {DeployInfraFormValue, DeploymentSize, InfraDeploymentRequestPayload, InfraDetails, InstallerState} from '../../types/installer.types';

@Component({
  selector: 'app-step-deploy-infra',
  standalone: true,
  imports: [
    CommonModule, ReactiveFormsModule, MatButtonModule, MatIconModule,
    MatProgressSpinnerModule, MatFormFieldModule, MatInputModule,
    MatSelectModule, MatCheckbox
  ],
  templateUrl: './step-deploy-infra.component.html',
  styleUrls: ['./step-deploy-infra.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepDeployInfraComponent implements OnInit, OnDestroy {
  @Output() nextStep = new EventEmitter<void>();
  @Output() previousStep = new EventEmitter<void>();
  @ViewChild('logContainer') private logContainer!: ElementRef;

  installerState!: InstallerState;
  deployInfraForm!: FormGroup;
  deploymentTypes: DeploymentSize[] = ['small', 'medium', 'large'];

  private unsubscribe$ = new Subject<void>();

  constructor(
    private installerStateService: InstallerStateService,
    private cdr: ChangeDetectorRef,
    private webSocketService: WebSocketService,
    private fb: FormBuilder,
    private router: Router,
    private apiService: ApiService
  ) { }

  ngOnInit(): void {
    const currentState = this.installerStateService.getCurrentState();
    this.deployInfraForm = this.fb.group({
      appName: [
        currentState.appName || '', [Validators.required, appNameValidator(6)]
      ],
      deploymentSize: [currentState.deploymentSize || '', Validators.required],
      enableCloudArmor: [currentState.enableCloudArmor || false],
      cloudArmorRateLimit: [
        currentState.cloudArmorRateLimit || 100,
        currentState.enableCloudArmor ?
            [Validators.required, Validators.min(1)] :
            []
      ]
    });

    this.deployInfraForm.get('enableCloudArmor')?.valueChanges
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe((enabled: boolean) => {
        const rateLimitCtrl = this.deployInfraForm.get('cloudArmorRateLimit');

        if (enabled) {
          rateLimitCtrl?.setValidators(
              [Validators.required, Validators.min(1)]);
        } else {
          rateLimitCtrl?.clearValidators();
        }
        rateLimitCtrl?.updateValueAndValidity();
      });

    this.installerStateService.installerState$
      .pipe(takeUntil(this.unsubscribe$))
      .subscribe((state: InstallerState) => {
        this.installerState = state;
        if (state.isConfigLocked) {
          this.deployInfraForm.get('appName')?.disable({emitEvent: false});
          this.deployInfraForm.get('enableCloudArmor')?.disable({
            emitEvent: false
          });
          this.deployInfraForm.get('cloudArmorRateLimit')?.disable({
            emitEvent: false
          });
          this.deployInfraForm.get('deploymentSize')?.disable({
            emitEvent: false
          });
        }
        this.cdr.detectChanges();
        this.scrollToBottom();
      });

    this.deployInfraForm.valueChanges
        .pipe(
            takeUntil(this.unsubscribe$), debounceTime(300),
            distinctUntilChanged(
                (prev: DeployInfraFormValue, curr: DeployInfraFormValue) =>
                    JSON.stringify(prev) === JSON.stringify(curr)))
        .subscribe(() => {
          const rawValue =
              sanitizeFormValues(this.deployInfraForm.getRawValue());
          this.installerStateService.updateAppInfraConfig(
              rawValue.appName, rawValue.deploymentSize,
              rawValue.enableCloudArmor, Number(rawValue.cloudArmorRateLimit));
        });
  }

  ngOnDestroy(): void {
    this.webSocketService.closeConnection();
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

  public onDeployInfra(): void {
    this.deployInfraForm.markAllAsTouched();
    if (this.deployInfraForm.invalid &&
        !this.deployInfraForm.get('appName')?.disabled) {
      console.warn('Deploy Infra form is invalid. Submission halted.');
      return;
    }

    const currentState = this.installerStateService.getCurrentState();
    const gcpConfiguration = currentState.gcpConfiguration;

    if (!gcpConfiguration || !gcpConfiguration.projectId || !gcpConfiguration.region) {
      console.error('GCP configuration (projectId or region) is missing. Cannot proceed with deployment.');
      return;
    }

    if (this.installerState.deploymentStatus === 'in-progress' ||
        this.installerState.deploymentStatus === 'completed') {
      console.warn('Infrastructure deployment already in progress or completed.');
      return;
    }

    this.installerStateService.updateDeploymentStatus('in-progress');
    this.installerStateService.clearDeploymentLogs();
    this.installerStateService.setInfraDetails({}); // Clear previous infra details
    this.installerStateService.setAppExternalIp(null); // Clear previous app external IP
    this.installerStateService.addDeploymentLog('Initiating infrastructure deployment...');
    this.cdr.detectChanges();

    const formValues = sanitizeFormValues(this.deployInfraForm.getRawValue());

    const deployPayload: InfraDeploymentRequestPayload = {
      project_id: gcpConfiguration.projectId,
      region: gcpConfiguration.region,
      app_name: formValues.appName,
      type: formValues.deploymentSize,
      components: {
        gateway: this.installerState.deploymentGoal.gateway ||
            this.installerState.deploymentGoal.all || false,
        registry: this.installerState.deploymentGoal.registry ||
            this.installerState.deploymentGoal.all || false,
        bap: this.installerState.deploymentGoal.bap ||
            this.installerState.deploymentGoal.all || false,
        bpp: this.installerState.deploymentGoal.bpp ||
            this.installerState.deploymentGoal.all || false
      },
      enable_cloud_armor: formValues.enableCloudArmor || false,
      rate_limit_count: Number(formValues.cloudArmorRateLimit) || 100
    };

    console.log('Final Infrastructure Deployment Payload:', deployPayload);

    this.webSocketService.connect('ws://127.0.0.1:8000/ws/deployInfra')
      .pipe(
        catchError(error => {
          console.error('WebSocket connection error for infra deployment:', error);
          this.installerStateService.updateDeploymentStatus('failed');
          this.installerStateService.addDeploymentLog(`WebSocket connection error: ${error.message || 'Could not connect to backend server.'}`);
          this.cdr.detectChanges();
          this.webSocketService.closeConnection();
          return EMPTY;
        }),
        finalize(() => {
          console.log('WebSocket stream for infra deployment finalized.');
          if (this.installerState.deploymentStatus === 'in-progress') {
            this.installerStateService.updateDeploymentStatus('failed');
            this.installerStateService.addDeploymentLog('Deployment failed: WebSocket disconnected unexpectedly or stream completed prematurely.');
          }
          this.cdr.detectChanges();
        }),
        takeUntil(this.unsubscribe$)
      )
      .subscribe({
        next: (message) => this.handleWebSocketMessage(message),
        error: (err) => {
          console.error('WebSocket runtime error during infra deployment:', err);
          this.installerStateService.updateDeploymentStatus('failed');
          this.installerStateService.addDeploymentLog(`Deployment failed: ${err.message || 'An unknown error occurred during deployment.'}`);
          this.cdr.detectChanges();
          this.webSocketService.closeConnection();
        },
        complete: () => {
          console.log('WebSocket connection for infra deployment closed by server.');
          this.cdr.detectChanges();
        }
      });

    this.webSocketService.sendMessage(deployPayload);
  }

  public hasAppNameError(): boolean {
    const ctrl = this.deployInfraForm.get('appName');
    return (ctrl?.hasError('invalidAppName') &&
            (ctrl?.touched || ctrl?.dirty)) ??
        false;
  }

  public hasRateLimitError(): boolean {
    const ctrl = this.deployInfraForm.get('cloudArmorRateLimit');
    return (ctrl?.invalid && (ctrl?.touched || ctrl?.dirty)) ?? false;
  }


  private handleWebSocketMessage(message: any): void {
    let parsedMessage: any;
    try {
      parsedMessage = typeof message === 'string' ? JSON.parse(message) : message;
    } catch (e) {
      console.warn('Received non-JSON WebSocket message for infra deployment:', message);
      this.installerStateService.addDeploymentLog(String(message));
      this.cdr.detectChanges();
      return;
    }

    console.log('Received parsed message for infra deployment:', parsedMessage);

    switch (parsedMessage.type) {
      case 'log':
        this.installerStateService.addDeploymentLog(parsedMessage.message);
        break;
      case 'success':
        this.installerStateService.updateDeploymentStatus('completed');
        this.installerStateService.addDeploymentLog('Infrastructure Deployment Completed Successfully!');

        const infraOutputs: InfraDetails = parsedMessage.message;
        this.installerStateService.setInfraDetails(infraOutputs);

        // Update appExternalIp logic to read from 'global_ip_address' as per logs
        if (infraOutputs && infraOutputs['global_ip_address'] && infraOutputs['global_ip_address'].value) {
            this.installerStateService.setAppExternalIp(infraOutputs['global_ip_address'].value);
            this.installerStateService.addDeploymentLog(`Application External IP: ${infraOutputs['global_ip_address'].value}`);
        } else {
            console.warn('Could not find global_ip_address in infraOutputs. Frontend might not display IP.');
            this.installerStateService.setAppExternalIp(null); // Explicitly set to null if not found
        }

        this.webSocketService.closeConnection();
        break;
      case 'error':
        this.installerStateService.updateDeploymentStatus('failed');
        this.installerStateService.addDeploymentLog(`Infrastructure Deployment Failed: ${parsedMessage.message || 'Unknown error.'}`);
        this.webSocketService.closeConnection();
        break;
      default:
        this.installerStateService.addDeploymentLog(`[UNKNOWN MESSAGE TYPE FOR INFRA DEPLOYMENT] ${JSON.stringify(parsedMessage)}`);
        break;
    }
    this.cdr.detectChanges();
    this.scrollToBottom();
  }

  private scrollToBottom(): void {
    try {
      if (this.logContainer && this.logContainer.nativeElement) {
        this.logContainer.nativeElement.scrollTop = this.logContainer.nativeElement.scrollHeight;
      }
    } catch (err) {
      console.error('Could not scroll to bottom:', err);
    }
  }

  getDeploymentTypeDisplay(type: DeploymentSize): string {
    switch (type) {
      case 'small':
        return 'Small - 50 tps';
      case 'medium':
        return 'Medium - 500 tps';
      case 'large':
        return 'Large - 1000 tps';
      default:
        return type;
    }
  }

  trackByLog(index: number, log: string): number {
    return index;
  }

  /**
   * Formats a camelCase or snake_case key into a user-friendly title case string.
   * e.g., "cluster_name" -> "Cluster Name", "registryUrl" -> "Registry Url"
   * @param key The key string from outputs.json
   * @returns Formatted display string
   */
  formatOutputKey(key: string): string {
    // Replace underscores with spaces, then apply title case
    return key
      .replace(/_/g, ' ')
      .replace(/([A-Z])/g, ' $1')
      .replace(/\b\w/g, (char) => char.toUpperCase())
      .trim();
  }

  onNext(): void {
    if (this.installerState.deploymentStatus === 'completed') {
      this.router.navigate(['installer', 'domain-configuration']);
    } else {
      console.warn('Cannot proceed to next step, infrastructure deployment is not completed.');
    }
  }

  onBack(): void {
    this.router.navigate(['installer', 'gcp-connection']);
  }

  get isDeploying(): boolean {
    return this.installerState.deploymentStatus === 'in-progress';
  }

  get deploymentComplete(): boolean {
    return this.installerState.deploymentStatus === 'completed';
  }

  get deploymentFailed(): boolean {
    return this.installerState.deploymentStatus === 'failed';
  }
}