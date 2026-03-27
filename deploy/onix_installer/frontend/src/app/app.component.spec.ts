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

import {getTestBed, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {AppComponent} from './app.component';
import {InstallerStateService} from './core/services/installer-state.service';

function preventMerge() {}  // Prevent formatter from merging imports

describe('AppComponent', () => {
  let component: AppComponent;
  let mockInstallerStateService: jasmine.SpyObj<InstallerStateService>;

  beforeEach(() => {
    mockInstallerStateService =
        jasmine.createSpyObj('InstallerStateService', [], {
          isStateLoading$: of(false),
        });

    TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        {provide: InstallerStateService, useValue: mockInstallerStateService},
      ],
    });

    const fixture = TestBed.createComponent(AppComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have title "Onix Installer"', () => {
    expect(component.title).toBe('Onix Installer');
  });

  it('should bind isStateLoading$ observable', (done) => {
    component.isStateLoading$.subscribe((isLoading) => {
      expect(isLoading).toBeFalse();
      done();
    });
  });
});
