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
import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';

import {InstallerStepGuard} from '../core/guards/installer-step.guard';

import {InstallerComponent} from './installer.component';
import {StepAppDeployComponent} from './steps/step_deploy_app/step-deploy-app.component';
import {StepDeployInfraComponent} from './steps/step_deploy_infra/step-deploy-infra.component';
import {StepDomainConfigComponent} from './steps/step_domain_configuration/step-domain-configuration.component';
import {StepGcpConnectionComponent} from './steps/step_gcp_connection/step-gcp-connection.component';
import {StepGoalComponent} from './steps/step_goal/step-goal.component';
import {StepHealthCheck} from './steps/step_health_check/step-health-check';
import {StepPrerequisitesComponent} from './steps/step_prerequisites/step-prerequisites.component';
import {StepSubscribe} from './steps/step_subscribe/step-subscribe';
// Import the new step component
import {StepWelcomeComponent} from './steps/step_welcome/step-welcome.component';

const routes: Routes = [
  {
    path: '',
    component: InstallerComponent,
    children: [
      { path: 'welcome', component: StepWelcomeComponent },
      { path: 'goal', component: StepGoalComponent},
      { path: 'prerequisites', component: StepPrerequisitesComponent},
      { path: 'gcp-connection', component: StepGcpConnectionComponent },
      { path: 'deploy-infra', component: StepDeployInfraComponent },
      { path: 'domain-configuration', component: StepDomainConfigComponent},
      { path: 'deploy-app', component: StepAppDeployComponent},
      { path: 'health-checks', component: StepHealthCheck },
      { path: 'subscribe', component: StepSubscribe },

    //   { path: 'summary', component: undefined /* StepSummaryComponent */ /*, canActivate: [InstallerStepGuard] */ },
      { path: '', redirectTo: 'welcome', pathMatch: 'full' }, // Redirect to welcome by default
      { path: '**', redirectTo: 'welcome' } // Catch-all
    ]
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule],
})
export class InstallerRoutingModule {}