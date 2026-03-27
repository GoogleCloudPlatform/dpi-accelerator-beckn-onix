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

import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule, FormControl } from '@angular/forms';

import { MatButtonModule } from '@angular/material/button';
import { MatRadioModule } from '@angular/material/radio';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';

import { InstallerStateService } from '../../../core/services/installer-state.service';
import { InstallerGoal, DeploymentGoal } from '../../types/installer.types';
import { Subscription, combineLatest } from 'rxjs';
import { startWith } from 'rxjs/operators';

@Component({
  selector: 'app-step-goal',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatRadioModule,
    MatFormFieldModule,
    MatCheckboxModule,
    MatIconModule,
    MatDividerModule,
  ],
  templateUrl: './step-goal.component.html',
  styleUrls: ['./step-goal.component.css']
})
export class StepGoalComponent implements OnInit, OnDestroy {
  //  NOTE: Create a base class for steps to reduce code duplication, especially for navigation and form handling.
  goalForm!: FormGroup;
  createNetworkComponentsForm!: FormGroup;
  joinNetworkComponentsForm!: FormGroup;

  selectedInstallerGoal: InstallerGoal | null = null;
  isGoalStepValid: boolean = false;
  isReadOnly:boolean=false;

  private installerGoalSubscription: Subscription | undefined;
  private createComponentsSubscription: Subscription | undefined;
  private joinComponentsSubscription: Subscription | undefined;
  private formValidationSubscription: Subscription | undefined;


  constructor(
    private fb: FormBuilder,
    private router: Router,
    private installerStateService: InstallerStateService,
    private cdr: ChangeDetectorRef
  ) { }

  ngOnInit(): void {
    this.goalForm = this.fb.group({
      installerGoal: [null as InstallerGoal | null, Validators.required],
    });

    this.createNetworkComponentsForm = this.fb.group({
      registry: [false],
      gateway: [false],
      bap: [false],
      bpp: [false],
      all: [false]
    });

    this.joinNetworkComponentsForm = this.fb.group({
      bap: [false],
      bpp: [false],
      all: [false]
    });

    const currentState = this.installerStateService.getCurrentState();
   if (currentState.deploymentStatus === 'completed' || currentState.deploymentStatus === 'failed') {
      this.isReadOnly = true;
      this.goalForm.disable();
      this.createNetworkComponentsForm.disable();
      this.joinNetworkComponentsForm.disable();
    }

    if (currentState.installerGoal) {
      this.goalForm.get('installerGoal')?.setValue(currentState.installerGoal, { emitEvent: false });
      this.selectedInstallerGoal = currentState.installerGoal;
    }

    if (currentState.deploymentGoal) {
      if (currentState.installerGoal === 'create_new_open_network') {
        const { all, ...rest } = currentState.deploymentGoal;
        this.createNetworkComponentsForm.patchValue(rest, { emitEvent: false });
        this._updateAllCheckboxState(this.createNetworkComponentsForm, ['registry', 'gateway', 'bap', 'bpp']);
      } else if (currentState.installerGoal === 'join_existing_network') {
        const { all, ...rest } = currentState.deploymentGoal;
        this.joinNetworkComponentsForm.patchValue(rest, { emitEvent: false });
        this._updateAllCheckboxState(this.joinNetworkComponentsForm, ['bap', 'bpp']);
      }
    }

    this.installerGoalSubscription = this.goalForm.get('installerGoal')?.valueChanges.subscribe((goal: InstallerGoal) => {
      this.selectedInstallerGoal = goal;
      this.resetComponentForms(goal);
      this.updateNextButtonState();
    });

    this.createComponentsSubscription = this.createNetworkComponentsForm.valueChanges.subscribe(value => {
      this._updateAllCheckboxState(this.createNetworkComponentsForm, ['registry', 'gateway', 'bap', 'bpp']);
      this.updateNextButtonState();
    });

    this.joinComponentsSubscription = this.joinNetworkComponentsForm.valueChanges.subscribe(value => {
      this._updateAllCheckboxState(this.joinNetworkComponentsForm, ['bap', 'bpp']);
      this.updateNextButtonState();
    });

    this.updateNextButtonState();

    this.formValidationSubscription = combineLatest([
      this.goalForm.valueChanges.pipe(startWith(this.goalForm.value)),
      this.createNetworkComponentsForm.valueChanges.pipe(startWith(this.createNetworkComponentsForm.value)),
      this.joinNetworkComponentsForm.valueChanges.pipe(startWith(this.joinNetworkComponentsForm.value))
    ]).subscribe(() => {
      this.updateNextButtonState();
    });
  }

  ngOnDestroy(): void {
    this.installerGoalSubscription?.unsubscribe();
    this.createComponentsSubscription?.unsubscribe();
    this.joinComponentsSubscription?.unsubscribe();
    this.formValidationSubscription?.unsubscribe();
  }

  toggleSelectAll(event: any, formType: 'create' | 'join'): void {
    const checked = event.checked;

    let targetFormGroup: FormGroup;
    let componentControlNames: string[];

    if (formType === 'create') {
      targetFormGroup = this.createNetworkComponentsForm;
      componentControlNames = ['registry', 'gateway', 'bap', 'bpp'];
    } else {
      targetFormGroup = this.joinNetworkComponentsForm;
      componentControlNames = ['bap', 'bpp'];
    }

    componentControlNames.forEach(controlName => {
      targetFormGroup.controls[controlName].setValue(!checked, { emitEvent: false });
      console.log(`Set control "${controlName}" to:`, checked);
    });

    targetFormGroup.updateValueAndValidity();
    this.cdr.detectChanges();
  }

  private _updateAllCheckboxState(formGroup: FormGroup, componentControlNames: string[]): void {
    const currentAllValue = formGroup.controls['all'].value;
    const allIndividualsChecked = componentControlNames.every(name => formGroup.controls[name].value);

    if (currentAllValue !== allIndividualsChecked) {
      formGroup.controls['all'].setValue(allIndividualsChecked, { emitEvent: false });
    }
  }

  private resetComponentForms(goal: InstallerGoal): void {
    if (goal === 'create_new_open_network') {
      this.joinNetworkComponentsForm.reset({ bap: false, bpp: false, all: false }, { emitEvent: false });
      this.createNetworkComponentsForm.updateValueAndValidity();
    } else {
      this.createNetworkComponentsForm.reset({ registry: false, gateway: false, bap: false, bpp: false, all: false }, { emitEvent: false });
      this.joinNetworkComponentsForm.updateValueAndValidity();
    }
  }

  private updateNextButtonState(): void {
    const installerGoal = this.goalForm.get('installerGoal')?.value;
    let componentsSelected = false;

    if (installerGoal === 'create_new_open_network') {
      const { registry, gateway, bap, bpp } = this.createNetworkComponentsForm.value;
      componentsSelected = registry || gateway || bap || bpp;
    } else if (installerGoal === 'join_existing_network') {
      const { bap, bpp } = this.joinNetworkComponentsForm.value;
      componentsSelected = bap || bpp;
    }

    this.isGoalStepValid = this.goalForm.valid && componentsSelected;
  }


  goToPreviousStep(): void {
    const currentInstallerGoal = this.goalForm.get('installerGoal')?.value;
    let currentDeploymentGoal: DeploymentGoal | undefined;

    if (currentInstallerGoal === 'create_new_open_network') {
      const { all, ...rest } = this.createNetworkComponentsForm.value;
      currentDeploymentGoal = rest;
    } else if (currentInstallerGoal === 'join_existing_network') {
      const { all, ...rest } = this.joinNetworkComponentsForm.value;
      currentDeploymentGoal = rest;
    }

    this.installerStateService.updateState({
      installerGoal: currentInstallerGoal,
      deploymentGoal: currentDeploymentGoal
    });
    this.router.navigate(['installer', 'welcome']);
  }

  goToNextStep(): void {
    this.goalForm.markAllAsTouched();
    if (this.selectedInstallerGoal === 'create_new_open_network') {
      this.createNetworkComponentsForm.markAllAsTouched();
    } else if (this.selectedInstallerGoal === 'join_existing_network') {
      this.joinNetworkComponentsForm.markAllAsTouched();
    }

    this.updateNextButtonState();

    if (this.isGoalStepValid) {
      const installerGoal = this.goalForm.value.installerGoal;
      let deploymentGoal: DeploymentGoal = {
        registry: false,
        gateway: false,
        bap: false,
        bpp: false,
      };

      if (installerGoal === 'create_new_open_network') {
        const { all, ...rest } = this.createNetworkComponentsForm.value;
        deploymentGoal = rest;
      } else if (installerGoal === 'join_existing_network') {
        const { all, ...rest } = this.joinNetworkComponentsForm.value;
        deploymentGoal = { ...rest, registry: false, gateway: false };
      }

      this.installerStateService.updateInstallerGoal(installerGoal);
      this.installerStateService.updateDeploymentGoal(deploymentGoal);

      this.router.navigate(['installer', 'prerequisites']);
    } else {
      alert('Please select a goal and at least one component to deploy.');
    }
  }
}
