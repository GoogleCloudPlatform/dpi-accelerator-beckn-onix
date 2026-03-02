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

import {StepperSelectionEvent} from '@angular/cdk/stepper';
import {CommonModule} from '@angular/common';
import {AfterViewInit, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild, ChangeDetectionStrategy,} from '@angular/core';
import {MatStepper} from '@angular/material/stepper';
import {ActivatedRoute, Event, NavigationEnd, Router, RouterModule} from '@angular/router';
import {Subscription} from 'rxjs';
import {filter, startWith} from 'rxjs/operators';

import {InstallerStateService} from '../core/services/installer-state.service';
import {SharedModule} from '../shared/shared.module';

import {InstallerConstants} from './constants/installer-constants';
import {InstallerState} from './types/installer.types';

@Component({
  changeDetection: ChangeDetectionStrategy.Eager,selector: 'app-installer',
  templateUrl: './installer.component.html',
  styleUrls: ['./installer.component.css'],
  standalone: true,
  imports: [CommonModule, RouterModule, SharedModule],
})
export class InstallerComponent implements OnInit, OnDestroy, AfterViewInit {
  @ViewChild('stepper') stepper!: MatStepper;

  title = 'Onix 2.0 Installer';
  steps: string[] = InstallerConstants.STEPS;
  stepPaths: string[] = InstallerConstants.STEP_PATHS;
  installerState!: InstallerState;
  isDeploying: boolean = false;

  private routerSubscription: Subscription | undefined;
  private stateSubscription: Subscription | undefined;

  constructor(
    private router: Router,
    private cdr: ChangeDetectorRef,
    private installerStateService: InstallerStateService
  ) {}

  ngOnInit(): void {
    // Subscribe to the installer state
    this.stateSubscription = this.installerStateService.installerState$.subscribe((state: InstallerState) => {
      this.installerState = state;
      this.isDeploying = state.deploymentStatus === 'in-progress';

      if (this.stepper && this.stepper.selectedIndex !== state.currentStepIndex) {
        this.stepper.selectedIndex = state.currentStepIndex;
      }
      this.cdr.detectChanges();
    });

    this.routerSubscription =
        this.router.events
            .pipe(
                filter(
                    (event: Event): event is NavigationEnd =>
                        event instanceof NavigationEnd),
                startWith(this.router))
            .subscribe((event: NavigationEnd|Router) => {
              const url = (event instanceof NavigationEnd) ? event.url :
                                                             this.router.url;
              const currentPath = url.split('/').pop() || '';
              const foundIndex = this.stepPaths.indexOf(currentPath);

              if (foundIndex !== -1 &&
                  foundIndex !== this.installerState.currentStepIndex) {
                this.installerStateService.updateCurrentStep(foundIndex);
              }
            });
  }

  ngAfterViewInit(): void {
    if (this.installerState && this.stepper) {
        this.stepper.selectedIndex = this.installerState.currentStepIndex;
        this.cdr.detectChanges();
    }
  }

  ngOnDestroy(): void {
    this.routerSubscription?.unsubscribe();
    this.stateSubscription?.unsubscribe();
  }

  onStepChange(event: StepperSelectionEvent): void {
    if (this.isDeploying) {
      this.stepper.selectedIndex = event.previouslySelectedIndex;
      return;
    }

    const nextRoute = this.stepPaths[event.selectedIndex];
    if (nextRoute) {
      this.router.navigate(['installer', nextRoute]);
    }
  }

  isStepCompleted(index: number): boolean {
    if (!this.installerState) {
      return false;
    }
    // NOTE: Refactor this logic to use explicit status flags for each step
    return index < this.installerState.highestStepReached;
  }
}