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

import {LoadingSpinnerComponent} from './loading-spinner.component';

// Prevent js_scrub from stripping the import if it thinks it's unused in simple
// analysis
const _dummyLoadingSpinnerComponent = LoadingSpinnerComponent;

describe('LoadingSpinnerComponent', () => {
  let component: LoadingSpinnerComponent;
  let fixture: ComponentFixture<LoadingSpinnerComponent>;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [LoadingSpinnerComponent],
        })
        .compileComponents();

    fixture = TestBed.createComponent(LoadingSpinnerComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should have default values', () => {
    expect(component.diameter).toBe(50);
    expect(component.strokeWidth).toBe(5);
    expect(component.color).toBe('primary');
    expect(component.message).toBe('Loading...');
  });

  it('should allow setting inputs', () => {
    component.diameter = 100;
    component.strokeWidth = 10;
    component.color = 'accent';
    component.message = 'Processing...';

    fixture.detectChanges();

    expect(component.diameter).toBe(100);
    expect(component.strokeWidth).toBe(10);
    expect(component.color).toBe('accent');
    expect(component.message).toBe('Processing...');
  });
});
