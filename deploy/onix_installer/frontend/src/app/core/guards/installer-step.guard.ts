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

import { Injectable } from '@angular/core';
import {
  CanActivate,
  ActivatedRouteSnapshot,
  Router,
  UrlTree,
} from '@angular/router';
import { Observable, of } from 'rxjs';
import { InstallerStateService } from '../services/installer-state.service';

@Injectable({
  providedIn: 'root',
})
export class InstallerStepGuard implements CanActivate {
  constructor(
    private installerStateService: InstallerStateService,
    private router: Router
  ) {}

  canActivate(
    route: ActivatedRouteSnapshot
  ): Observable<boolean | UrlTree> {
    const targetStepPath = route.url[0]?.path;
    const currentState = this.installerStateService.getCurrentState();
    const stepOrder = [
        'welcome',
        'goal',
        'prerequisites',
        'gcp-connection',
        'deploy-infra',
        'domain-configuration',
        'deploy-app',
        'health-checks',
        'subscribe'
    ];

    const targetIndex = stepOrder.indexOf(targetStepPath);
    if (targetIndex <= currentState.highestStepReached) {
        return of(true);
    }
    switch (targetStepPath) {
      case 'prerequisites':
        if (!currentState.installerGoal) {
          return of(this.router.createUrlTree(['/installer/goal']));
        }
        break;

      case 'gcp-connection':
        if (!currentState.prerequisitesMet) {
          return of(this.router.createUrlTree(['/installer/prerequisites']));
        }
        break;

      case 'deploy-infra':
        if (!currentState.gcpConfiguration?.projectId || !currentState.gcpConfiguration?.region) {
          return of(this.router.createUrlTree(['/installer/gcp-connection']));
        }
        break;

      case 'domain-configuration':
        if (currentState.deploymentStatus !== 'completed' || !currentState.infraDetails) {
          return of(this.router.createUrlTree(['/installer/deploy-infra']));
        }
        break;

      case 'deploy-app':
        if (!currentState.globalDomainConfig || currentState.subdomainConfigs.length === 0) {
          return of(this.router.createUrlTree(['/installer/domain-configuration']));
        }
        break;

      case 'health-checks':
        if (currentState.appDeploymentStatus !== 'completed' || Object.keys(currentState.deployedServiceUrls).length === 0) {
          return of(this.router.createUrlTree(['/installer/deploy-app']));
        }
        break;
    }
    return of(true);
  }
}
