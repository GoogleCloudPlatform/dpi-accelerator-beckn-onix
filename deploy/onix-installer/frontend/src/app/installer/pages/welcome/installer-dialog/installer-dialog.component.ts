import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormArray } from '@angular/forms';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { ReactiveFormsModule, FormGroup, FormBuilder, Validators, AbstractControl, ValidatorFn } from '@angular/forms';
import { MatListModule } from '@angular/material/list';
import { Subscription } from 'rxjs';

import { WelcomeStepComponent } from '../welcome-step/welcome-step.component';
import { DeploymentIdentifiersStepComponent } from '../deployment-identifiers-step/deployment-identifiers-step.component';
import { GoalStepComponent } from '../goal-step/goal-step.component';
import { ConfigDeployStepComponent } from '../config-deploy-step/config-deploy-step.component';

interface WizardStep {
  label: string;
  form: FormGroup;
  isComplete: boolean;
}

export function requireAtLeastOneCheckboxValidator(): ValidatorFn {
  return (control: AbstractControl): {[key: string]: any} | null => {
    const formGroup = control as FormGroup;
    if (!formGroup.controls) return null;
    for (const key in formGroup.controls) {
      if (formGroup.controls[key].value === true) {
        return null;
      }
    }
    return { 'requireAtLeastOneCheckbox': true };
  };
}

@Component({
  selector: 'app-installer-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    ReactiveFormsModule,
    MatListModule,
    WelcomeStepComponent,
    DeploymentIdentifiersStepComponent,
    GoalStepComponent,
  ],
  templateUrl: './installer-dialog.component.html',
  styleUrls: ['./installer-dialog.component.css']
})
export class InstallerDialogComponent implements OnInit, OnDestroy {
  currentStepIndex: number = 0;
  steps: WizardStep[] = [];
  isCurrentStepValid: boolean = false;
  private statusSubscription: Subscription | undefined;

  constructor(
    public dialogRef: MatDialogRef<InstallerDialogComponent>,
    private fb: FormBuilder
  ) { }

  ngOnInit(): void {
    const welcomeFormGroup = this.fb.group({
      isAcknowledged: [false, Validators.requiredTrue]
    });

    const identifiersFormGroup = this.fb.group({
      appName: ['', [Validators.required, Validators.minLength(3)]],
      appId: [{value: '', disabled: true}],
      projectTags: this.fb.array([
          this.fb.control('', Validators.required)
      ]),
      projectRegion: ['', [Validators.required]]
    });

    const goalFormGroup = this.fb.group({
      deploymentType: ['', Validators.required],
      newNetworkComponents: this.fb.group({
        registry: [false], gateway: [false], bap: [false], bpp: [false], all: [false]
      }, { validators: requireAtLeastOneCheckboxValidator() }),
      existingNetworkComponents: this.fb.group({
        bap: [false], bpp: [false], all: [false]
      }, { validators: requireAtLeastOneCheckboxValidator() })
    });

    goalFormGroup.get('newNetworkComponents')?.disable({ emitEvent: false });
    goalFormGroup.get('existingNetworkComponents')?.disable({ emitEvent: false });

    goalFormGroup.get('deploymentType')?.valueChanges.subscribe(value => {
        if (value === 'new') {
            goalFormGroup.get('newNetworkComponents')?.enable({ emitEvent: false });
            goalFormGroup.get('existingNetworkComponents')?.disable({ emitEvent: false });
             goalFormGroup.get('existingNetworkComponents')?.reset({}, { emitEvent: false });
        } else if (value === 'join') {
            goalFormGroup.get('newNetworkComponents')?.disable({ emitEvent: false });
            goalFormGroup.get('newNetworkComponents')?.reset({}, { emitEvent: false });
            goalFormGroup.get('existingNetworkComponents')?.enable({ emitEvent: false });
        } else {
            goalFormGroup.get('newNetworkComponents')?.disable({ emitEvent: false });
            goalFormGroup.get('existingNetworkComponents')?.disable({ emitEvent: false });
        }
        goalFormGroup.updateValueAndValidity();
    });

    identifiersFormGroup.get('appName')?.valueChanges.subscribe(name => {
      const slug = name ? name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') : '';
      identifiersFormGroup.get('appId')?.setValue(slug ? `onix-${slug}` : '');
    });

    const infraFormGroup = this.fb.group({
      deploymentType: ['gke', Validators.required], // Default to GKE
      trafficVolume: ['small', Validators.required]
    });

    this.steps = [
      { label: 'Welcome', form: welcomeFormGroup, isComplete: false },
      { label: 'Deployment Identifiers', form: identifiersFormGroup, isComplete: false },
      { label: 'Goal', form: goalFormGroup, isComplete: false },
      { label: 'Config & Deploy Infra', form: infraFormGroup, isComplete: false },
      { label: 'Domain Config', form: this.fb.group({}), isComplete: false },
      { label: 'Deploy App', form: this.fb.group({}), isComplete: false },
    ];

    this.subscribeToFormStatus(this.currentStepIndex);
  }

  private subscribeToFormStatus(index: number): void {
    this.statusSubscription?.unsubscribe();
    const currentForm = this.steps[index].form;
    this.isCurrentStepValid = currentForm.valid;
    this.statusSubscription = currentForm.statusChanges.subscribe(() => {
        this.isCurrentStepValid = currentForm.valid;
    });
  }

  closeDialog(): void {
    this.dialogRef.close();
  }

  onStepAcknowledged(index: number): void {
    if (index === this.currentStepIndex && this.steps[index].form.valid) {
      if(index < this.steps.length - 1) {
        this.steps[index].isComplete = true;
        this.goToNextStep();
      } else {
        // Handle Finish action
        this.closeDialog();
      }
    }
  }

  goToNextStep(): void {
    if (this.currentStepIndex < this.steps.length - 1) {
      this.currentStepIndex++;
      this.subscribeToFormStatus(this.currentStepIndex);
    }
  }

  goToPreviousStep(): void {
    if (this.currentStepIndex > 0) {
      this.currentStepIndex--;
      this.subscribeToFormStatus(this.currentStepIndex);
    }
  }

  setCurrentStep(index: number): void {
    if (index >= 0 && index < this.steps.length) {
      if (this.steps[index].isComplete || index < this.currentStepIndex || index === 0) {
         // Allow navigation to previous completed steps or the first step
        const isNavigatingForward = index > this.currentStepIndex;
        if (isNavigatingForward) {
          // Check if all intermediate steps are complete
          let canNavigate = true;
          for(let i = this.currentStepIndex; i < index; i++) {
            if (!this.steps[i].isComplete) {
              canNavigate = false;
              break;
            }
          }
          if (!canNavigate) return;
        }
        this.currentStepIndex = index;
        this.subscribeToFormStatus(index);
      } else if (index === this.currentStepIndex) {
        // Do nothing if clicking the current step
      } else if (index === this.currentStepIndex + 1 && this.steps[this.currentStepIndex].isComplete) {
        // Allow navigation to the very next step if current is complete
         this.currentStepIndex = index;
         this.subscribeToFormStatus(index);
      }
    }
  }

  saveConfig(): void {
    console.log('Configuration saved:', this.steps[3].form.value);
    // TODO: Add logic for saving
  }

  ngOnDestroy(): void {
    this.statusSubscription?.unsubscribe();
  }
}
