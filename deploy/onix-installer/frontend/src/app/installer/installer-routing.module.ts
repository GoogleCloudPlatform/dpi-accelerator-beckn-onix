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

// src/app/installer/installer-routing.module.ts
import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { InstallerComponent } from './installer.component';

// Import the new step component
import { WelcomeComponent } from './pages/welcome/welcome.component';
import { StepGoalComponent } from './steps/step-goal/step-goal.component';
import { StepPrerequisitesComponent } from './steps/step-prerequisites/step-prerequisites.component';
import { StepGcpConnectionComponent } from './steps/step-gcp-connection/step-gcp-connection.component';
import { StepDeployInfraComponent } from './steps/step-deploy-infra/step-deploy-infra.component';
import { StepDomainConfigComponent } from './steps/step-domain-configuration/step-domain-configuration.component';
import { StepAppDeployComponent} from './steps/step-deploy-app/step-deploy-app.component';
import { StepHealthCheck } from './steps/step-health-check/step-health-check';
import { StepSubscribe } from './steps/step-subscribe/step-subscribe';

const routes: Routes = [
  {
    path: 'welcome',
    component: WelcomeComponent, // new full-screen welcome page
  },
  {
    path: 'steps',
    component: InstallerComponent, // This component now only contains the steps
    children: [
      { path: 'goal', component: StepGoalComponent},
      { path: 'prerequisites', component: StepPrerequisitesComponent},
      { path: 'gcp-connection', component: StepGcpConnectionComponent },
      { path: 'deploy-infra', component: StepDeployInfraComponent },
      { path: 'domain-configuration', component: StepDomainConfigComponent},
      { path: 'deploy-app', component: StepAppDeployComponent},
      { path: 'health-checks', component: StepHealthCheck },
      { path: 'subscribe', component: StepSubscribe },
      { path: '', redirectTo: 'goal', pathMatch: 'full' }, // Redirect to the first step
    ]
  },
  { path: 'test', redirectTo: 'welcome', pathMatch: 'full' }, // Default to the new welcome page
  { path: '**', redirectTo: 'welcome' } // Catch-all for any other route
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule],
})
export class InstallerRoutingModule {}
