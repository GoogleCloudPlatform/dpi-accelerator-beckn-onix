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

import { Component, OnInit, Output, EventEmitter, ChangeDetectionStrategy, ChangeDetectorRef, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule, FormControl } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatIconModule } from '@angular/material/icon';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Router } from '@angular/router';

import { InstallerStateService } from '../../../core/services/installer-state.service';
import { GcpConfiguration } from '../../types/installer.types';
import { BehaviorSubject, EMPTY, Subject } from 'rxjs';
import { catchError, debounceTime, distinctUntilChanged, takeUntil } from 'rxjs/operators';
import { ApiService } from '../../../core/services/api.service';

@Component({
  selector: 'app-step-gcp-connection',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatIconModule,
    MatAutocompleteModule,
    MatProgressSpinnerModule,
    MatTooltipModule
  ],
  templateUrl: './step-gcp-connection.component.html',
  styleUrls: ['./step-gcp-connection.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepGcpConnectionComponent implements OnInit, OnDestroy {
  @Output() nextStep = new EventEmitter<void>();
  @Output() previousStep = new EventEmitter<void>();

  gcpConnectionForm!: FormGroup;

  gcpProjects: string[] = [];
  gcpRegions: string[] = [];
  isRegionsLoading: boolean = false;
  isProjectsLoading: boolean = false;
  apiError: string | null = null;

  projectSearchCtrl = new FormControl('');
  regionSearchCtrl = new FormControl('');

  filteredGcpProjects = new BehaviorSubject<string[]>([]);
  filteredGcpRegions = new BehaviorSubject<string[]>([]);

  private unsubscribe$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private installerStateService: InstallerStateService,
    private cdr: ChangeDetectorRef,
    private router: Router,
    private apiService: ApiService
  ) { }

  ngOnInit(): void {
    const currentState = this.installerStateService.getCurrentState();
    const initialGcpConfig = currentState.gcpConfiguration;

    this.gcpConnectionForm = this.fb.group({
      projectId: [initialGcpConfig?.projectId || '', Validators.required],
      region: [initialGcpConfig?.region || '', Validators.required]
    });


    this.fetchGcpRegions();
    this.fetchGcpProjects();

      if (currentState.deploymentStatus === 'completed' ) {
      this.gcpConnectionForm.disable();
    }
    this.projectSearchCtrl.valueChanges.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      takeUntil(this.unsubscribe$)
    ).subscribe(value => {
      this.filterProjects(value || '');
      this.cdr.detectChanges();
    });

    this.regionSearchCtrl.valueChanges.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      takeUntil(this.unsubscribe$)
    ).subscribe(value => {
      this.filterRegions(value || '');
      this.cdr.detectChanges();
    });

    this.gcpConnectionForm.valueChanges.pipe(takeUntil(this.unsubscribe$)).subscribe(() => {
      this.updateGcpConnectionState();
    });

    this.updateGcpConnectionState();
  }

  ngOnDestroy(): void {
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

  private updateGcpConnectionState(): void {
    const gcpConfig: GcpConfiguration = {
      projectId: this.gcpConnectionForm.get('projectId')?.value,
      region: this.gcpConnectionForm.get('region')?.value
    };
    this.installerStateService.updateGcpConfiguration(gcpConfig);
    this.cdr.detectChanges();
  }

  private fetchGcpRegions(): void {
    this.isRegionsLoading = true;
    this.apiError = null;
    this.apiService.getGcpRegions()
      .pipe(
        catchError(error => {
          console.error('API Error fetching regions:', error);
          this.apiError = 'Failed to load GCP regions. Please check the backend connection or CORS settings.';
          this.isRegionsLoading = false;
          this.cdr.detectChanges();
          return EMPTY;
        }),
        takeUntil(this.unsubscribe$)
      )
      .subscribe(regions => {
        console.log('API Response Received (Regions):', regions);
        this.gcpRegions = regions;
        this.filteredGcpRegions.next(this.gcpRegions.slice());
        const initialRegion = this.installerStateService.getCurrentState().gcpConfiguration?.region;
        if (initialRegion && this.gcpRegions.includes(initialRegion)) {
          this.gcpConnectionForm.get('region')?.setValue(initialRegion);
        } else if (initialRegion) {
          this.gcpConnectionForm.get('region')?.setValue('');
        }
        this.isRegionsLoading = false;
        this.cdr.detectChanges();
      });
  }

  private fetchGcpProjects(): void {
    this.isProjectsLoading = true;
    this.apiError = null;
    this.apiService.getGcpProjectNames()
      .pipe(
        catchError(error => {
          console.error('API Error fetching projects:', error);
          this.apiError = 'Failed to load GCP projects. Please check the backend connection or CORS settings.';
          this.isProjectsLoading = false;
          this.cdr.detectChanges();
          return EMPTY;
        }),
        takeUntil(this.unsubscribe$)
      )
      .subscribe(projects => {
        console.log('API Response Received (Projects):', projects);
        this.gcpProjects = projects;
        this.filteredGcpProjects.next(this.gcpProjects.slice());
        const initialProjectId = this.installerStateService.getCurrentState().gcpConfiguration?.projectId;
        if (initialProjectId && this.gcpProjects.includes(initialProjectId)) {
          this.gcpConnectionForm.get('projectId')?.setValue(initialProjectId);
        } else if (initialProjectId) {
          this.gcpConnectionForm.get('projectId')?.setValue('');
        }
        this.isProjectsLoading = false;
        this.cdr.detectChanges();
      });
  }

  private filterProjects(value: string): void {
    const filterValue = value.toLowerCase();
    this.filteredGcpProjects.next(
      this.gcpProjects.filter(project => project.toLowerCase().includes(filterValue))
    );
  }

  private filterRegions(value: string): void {
    const filterValue = value.toLowerCase();
    this.filteredGcpRegions.next(
      this.gcpRegions.filter(region => region.toLowerCase().includes(filterValue))
    );
  }
  clearProjectSearch() {
    this.projectSearchCtrl.setValue('');
  }

  clearRegionSearch() {
    this.regionSearchCtrl.setValue('');
  }

  stopProp(event: Event) {
    event.stopPropagation();
  }

  onNext(): void {
    this.gcpConnectionForm.markAllAsTouched();
    this.updateGcpConnectionState();
    if (this.gcpConnectionForm.valid) {
      this.nextStep.emit();
      this.router.navigate(['installer', 'deploy-infra']);
    } else {
      console.log('Form is invalid:', this.gcpConnectionForm.errors);
    }
  }

  onBack(): void {
    this.previousStep.emit();
    this.router.navigate(['installer', 'prerequisites']);
  }
}