/**
 * Copyright 2025 Google LLC
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

import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { Router } from '@angular/router';

@Component({
  selector: 'app-step-welcome',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule
  ],
  templateUrl: './step-welcome.component.html',
  styleUrls: ['./step-welcome.component.css']
})
export class StepWelcomeComponent {

  constructor(private router: Router) { }

  /**
   * Navigates to the deployment creation flow or installer.
   * Assuming the old 'goal' step is now the first step of the new deployment flow.
   */
  goToCreateDeployment(): void {
    // Change the route to the start of the new process
    // You might want to change 'installer/goal' to something like 'deployment/new'
    this.router.navigate(['installer', 'goal']); 
    // If you're removing the installer wizard entirely, ensure your main routing
    // points the base path ('/') to this new component.
  }
}