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
import { FormBuilder, FormGroup, FormArray, FormControl, ReactiveFormsModule, Validators, AbstractControl } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { MatFormFieldModule } from '@angular/material/form-field'; // Import MatFormFieldModule for MatError usage
import { Router } from '@angular/router';
import { InstallerConstants } from '../../constants/installer-constants';
import { InstallerStateService } from '../../../core/services/installer-state.service';
import { Subscription } from 'rxjs';
@Component({
  selector: 'app-step-prerequisites',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatCheckboxModule,
    MatIconModule,
    MatDividerModule,
    MatFormFieldModule
  ],
  templateUrl: './step-prerequisites.component.html',
  styleUrls: ['./step-prerequisites.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepPrerequisitesComponent implements OnInit, OnDestroy {
  @Output() nextStep = new EventEmitter<void>();
  @Output() previousStep = new EventEmitter<void>();
  prerequisitesForm!: FormGroup;
  prerequisitesList = InstallerConstants.PREREQUISITES_LIST;
  allPrerequisitesMet: boolean = false;
  private formSubscription: Subscription | undefined;
  constructor(
    private fb: FormBuilder,
    private installerStateService: InstallerStateService,
    private cdr: ChangeDetectorRef,
    private router: Router
  ) { }

  ngOnInit(): void {
    this.prerequisitesForm = this.fb.group({
      items: this.fb.array([])
    });
    const prerequisitesFormArray = this.prerequisitesForm.get('items') as FormArray;
    const currentState = this.installerStateService.getCurrentState();
    const initialPrerequisitesMet = currentState.prerequisitesMet;
    const installerGoal = currentState.installerGoal;

    if (installerGoal === 'create_new_open_network') {
      this.prerequisitesList = InstallerConstants.PREREQUISITES_LIST.filter(
        prerequisite => !prerequisite.includes('Registry and Gateway URLs')
      );
    } else {
      this.prerequisitesList = InstallerConstants.PREREQUISITES_LIST;
    }

    this.prerequisitesList.forEach(() => {
      prerequisitesFormArray.push(this.fb.control(initialPrerequisitesMet));
    });
    prerequisitesFormArray.setValidators(this.allCheckboxesRequiredValidator());
    this.formSubscription = this.prerequisitesForm.valueChanges.subscribe(() => {
      this.updatePrerequisitesState();
    });
    this.updatePrerequisitesState();
    if (currentState.deploymentStatus === 'completed') {
      this.prerequisitesForm.disable();
    }
  }

  ngOnDestroy(): void {
    this.formSubscription?.unsubscribe();
  }

  private allCheckboxesRequiredValidator(): import('@angular/forms').ValidatorFn {
    return (control: AbstractControl): { [key: string]: boolean } | null => {
      if (control instanceof FormArray) {
        const allChecked = control.controls.every(ctrl => ctrl.value === true);
        return allChecked ? null : { allRequired: true };
      }
      return null;
    };
  }
  private updatePrerequisitesState(): void {
    const prerequisitesFormArray = this.prerequisitesForm.get('items') as FormArray;
    this.allPrerequisitesMet = prerequisitesFormArray.valid;
    this.installerStateService.updatePrerequisitesMet(this.allPrerequisitesMet);
    this.cdr.detectChanges();
  }

  getCheckboxControl(index: number): FormControl {
    return (this.prerequisitesForm.get('items') as FormArray).at(index) as FormControl;
  }
  onNext(): void {
    this.prerequisitesForm.markAllAsTouched();
    this.updatePrerequisitesState();
    if (this.prerequisitesForm.valid) {
      this.router.navigate(['installer', 'gcp-connection']);
    }
  }
  onBack(): void {
    this.router.navigate(['installer', 'goal']);
  }
}