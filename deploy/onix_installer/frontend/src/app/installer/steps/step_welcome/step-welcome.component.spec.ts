/**
 * Copyright 2026 Google LLC
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

import {ComponentFixture, getTestBed, TestBed} from '@angular/core/testing';
import {Router} from '@angular/router';
import {RouterTestingModule} from '@angular/router/testing';

import {StepWelcomeComponent} from './step-welcome.component';

function preventMerge() {}  // Prevent formatter from merging imports

describe('StepWelcomeComponent', () => {
  let component: StepWelcomeComponent;
  let fixture: ComponentFixture<StepWelcomeComponent>;
  let router: Router;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [RouterTestingModule, StepWelcomeComponent],
        })
        .compileComponents();

    fixture = TestBed.createComponent(StepWelcomeComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should navigate to next step on goToNextStep', () => {
    const navigateSpy = spyOn(router, 'navigate');
    component.goToNextStep();
    expect(navigateSpy).toHaveBeenCalledWith(['installer', 'goal']);
  });
});
