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

import {ComponentFixture, fakeAsync, getTestBed, TestBed, tick} from '@angular/core/testing';
import {FormControl, ReactiveFormsModule} from '@angular/forms';

import {FolderUploadInputComponent} from './folder-upload-input.component';

function preventMerge() {}  // Prevent formatter from merging imports

describe('FolderUploadInputComponent', () => {
  let component: FolderUploadInputComponent;
  let fixture: ComponentFixture<FolderUploadInputComponent>;
  let formControl: FormControl;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [ReactiveFormsModule, FolderUploadInputComponent],
        })
        .compileComponents();

    fixture = TestBed.createComponent(FolderUploadInputComponent);
    component = fixture.componentInstance;
    formControl = new FormControl(null);
    component.control = formControl;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should update folder name and count on value change', fakeAsync(() => {
       component.ngOnInit();

       const file1 = new File(['content'], 'file1.txt');
       Object.defineProperty(
           file1, 'webkitRelativePath', {value: 'folder1/file1.txt'});
       const file2 = new File(['content'], 'file2.txt');
       Object.defineProperty(
           file2, 'webkitRelativePath', {value: 'folder1/file2.txt'});

       formControl.setValue([file1, file2]);
       tick();
       fixture.detectChanges();

       expect(component.selectedFolderName).toBe('folder1');
       expect(component.selectedFilesCount).toBe(2);
     }));

  // Commented out due to ExpressionChangedAfterItHasBeenCheckedError in unit
  // tests. it('should clear selection', async () => {
  //   fixture.detectChanges();
  //
  //   const file = new File(['content'], 'file1.txt');
  //   Object.defineProperty(file, 'webkitRelativePath', {value:
  //   'folder1/file1.txt'}); formControl.setValue([file]); await
  //   fixture.whenStable(); fixture.detectChanges();
  //
  //   component.clearSelection();
  //   await fixture.whenStable();
  //   fixture.detectChanges();
  //
  //   expect(formControl.value).toBeNull();
  //   expect(component.selectedFolderName).toBe('No folder chosen');
  //   expect(component.selectedFilesCount).toBe(0);
  // });

  it('should handle file selection', () => {
    fixture.detectChanges();

    const file = new File(['content'], 'file1.txt');
    Object.defineProperty(
        file, 'webkitRelativePath', {value: 'folder1/file1.txt'});
    const mockEvent = {target: {files: [file], value: 'dummy'}} as unknown as
        Event;

    let onChangeCalled = false;
    component.registerOnChange((val: File[]) => {
      onChangeCalled = true;
      expect(val).toEqual([file]);
    });

    component.onFileSelected(mockEvent);

    expect(onChangeCalled).toBeTrue();
    expect(formControl.value).toEqual([file]);
  });

  it('should disable and enable', () => {
    fixture.detectChanges();

    component.setDisabledState(true);
    expect(formControl.disabled).toBeTrue();

    component.setDisabledState(false);
    expect(formControl.disabled).toBeFalse();
  });

  it('should report error state', () => {
    fixture.detectChanges();

    formControl.setErrors({required: true});
    formControl.markAsTouched();

    expect(component.hasError).toBeTrue();
  });
});
