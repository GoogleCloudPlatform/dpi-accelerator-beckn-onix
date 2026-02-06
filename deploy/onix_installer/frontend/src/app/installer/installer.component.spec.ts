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
import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {MatStepperModule} from '@angular/material/stepper';
import {BrowserDynamicTestingModule, platformBrowserDynamicTesting} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {NavigationEnd, Router} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';
import {BehaviorSubject, Subject} from 'rxjs';

import {InstallerStateService} from '../core/services/installer-state.service';
import {SharedModule} from '../shared/shared.module';

import {InstallerComponent} from './installer.component';
import {InstallerState} from './types/installer.types';

// A baseline mock state for the installer.
const mockInitialState: InstallerState = {
  currentStepIndex: 0,
  highestStepReached: 0,
  deploymentStatus: 'pending',
  appDeploymentStatus: 'pending',
  installerGoal: null,
  deploymentGoal: { bap: false, bpp: false, gateway: false, registry: false, all: false },
  prerequisitesMet: false,
  gcpConfiguration: null,
  appName: '',
  deploymentSize: 'small',
  infraDetails: null,
  appExternalIp: null,
  deployedServiceUrls: {},
  servicesDeployed: [],
  logsExplorerUrls: {},
  globalDomainConfig: null,
  componentSubdomainPrefixes: [],
  subdomainConfigs: [],
  dockerImageConfigs: [],
  appSpecificConfigs: [],
  healthCheckStatuses: [],
  deploymentLogs: [],
  appDeployImageConfig: null,
  appDeployRegistryConfig: null,
  appDeployGatewayConfig: null,
  appDeployAdapterConfig: null
};

// A mock InstallerStateService to control the component's state during tests.
class MockInstallerStateService {
  private state = new BehaviorSubject<InstallerState>(mockInitialState);
  installerState$ = this.state.asObservable();
  updateCurrentStep = jasmine.createSpy('updateCurrentStep');

  // Helper method to simulate state changes from other parts of the app.
  setState(newState: Partial<InstallerState>) {
    const currentState = this.state.getValue();
    this.state.next({ ...currentState, ...newState });
  }
}

describe('InstallerComponent', () => {
  let component: InstallerComponent;
  let fixture: ComponentFixture<InstallerComponent>;
  let installerStateService: MockInstallerStateService;
  let router: Router;
  let routerEvents: Subject<NavigationEnd>;

  beforeEach(async () => {
    routerEvents = new Subject<NavigationEnd>();

    await TestBed.configureTestingModule({
      imports: [
        InstallerComponent, // Standalone component
        RouterTestingModule.withRoutes([]),
        NoopAnimationsModule,
        MatStepperModule,
        SharedModule,
      ],
      providers: [
        { provide: InstallerStateService, useClass: MockInstallerStateService },
        {
          provide: Router,
          useValue: {
            navigate: jasmine.createSpy('navigate'),
            events: routerEvents.asObservable(),
            url: '/installer/welcome', // Initial URL
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(InstallerComponent);
    component = fixture.componentInstance;
    installerStateService = TestBed.inject(InstallerStateService) as any;
    router = TestBed.inject(Router);
  });

  it('should create the component', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  describe('Initialization (ngOnInit)', () => {
    it('should subscribe to installerState$ and update its state', () => {
      const testState: InstallerState = { ...mockInitialState, currentStepIndex: 2, deploymentStatus: 'in-progress' };
      installerStateService.setState(testState);

      fixture.detectChanges(); // ngOnInit

      expect(component.installerState).toEqual(testState);
      expect(component.isDeploying).toBeTrue();
    });

    it('should subscribe to router events and update the state service on navigation', fakeAsync(() => {
      fixture.detectChanges(); // ngOnInit
      tick();

      // Simulate navigating to the 'deploy-infra' step, which is at index 4
      (router as any).url = '/installer/deploy-infra';
      routerEvents.next(new NavigationEnd(1, '/installer/deploy-infra', '/installer/deploy-infra'));
      tick();

      expect(installerStateService.updateCurrentStep).toHaveBeenCalledWith(4);
    }));
  });

  describe('View Initialization and Destruction', () => {
    it('should sync stepper index in ngAfterViewInit', fakeAsync(() => {
      installerStateService.setState({ currentStepIndex: 2 });
      fixture.detectChanges();
      tick();
      expect(component.stepper.selectedIndex).toBe(0);
    }));

    it('should unsubscribe from all subscriptions on ngOnDestroy', () => {
      fixture.detectChanges();

      const stateSub = component['stateSubscription'];
      const routerSub = component['routerSubscription'];
      spyOn(stateSub!, 'unsubscribe');
      spyOn(routerSub!, 'unsubscribe');

      component.ngOnDestroy();

      expect(stateSub!.unsubscribe).toHaveBeenCalled();
      expect(routerSub!.unsubscribe).toHaveBeenCalled();
    });
  });

  describe('User Interaction (onStepChange)', () => {
    it('should navigate to the correct route when a step is changed', () => {
      fixture.detectChanges();
      const event: StepperSelectionEvent = { selectedIndex: 3, previouslySelectedIndex: 2 } as any;

      component.onStepChange(event);

      // stepPaths[3] is 'gcp-connection'
      expect(router.navigate).toHaveBeenCalledWith(['installer', 'gcp-connection']);
    });
  });

  describe('State Logic (isStepCompleted)', () => {
    it('should return true for steps with an index less than highestStepReached', () => {
      installerStateService.setState({ highestStepReached: 5 });
      fixture.detectChanges();

      expect(component.isStepCompleted(0)).toBeTrue();
      expect(component.isStepCompleted(4)).toBeTrue();
    });

    it('should return false for steps with an index equal to or greater than highestStepReached', () => {
      installerStateService.setState({ highestStepReached: 5 });
      fixture.detectChanges();

      expect(component.isStepCompleted(5)).toBeFalse();
      expect(component.isStepCompleted(6)).toBeFalse();
    });
  });
});