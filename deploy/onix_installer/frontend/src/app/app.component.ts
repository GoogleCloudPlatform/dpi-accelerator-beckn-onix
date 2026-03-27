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

import {CommonModule} from '@angular/common';
import {Component, ChangeDetectionStrategy} from '@angular/core';
import {RouterOutlet} from '@angular/router';
import {Observable} from 'rxjs';

import {InstallerStateService} from './core/services/installer-state.service';
import {LoadingSpinnerComponent} from './shared/components/loading_spinner/loading-spinner.component';

@Component({
  changeDetection: ChangeDetectionStrategy.Eager,selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css'],
  standalone: true,
  imports: [CommonModule, RouterOutlet, LoadingSpinnerComponent],
})
export class AppComponent {
  title = 'Onix Installer';
   isStateLoading$: Observable<boolean>;

   constructor(private installerStateService: InstallerStateService) {
    this.isStateLoading$ = this.installerStateService.isStateLoading$;
   }
}