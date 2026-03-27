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
import {FormControl} from '@angular/forms';

import {FileUploadInputComponent} from './file-upload-input.component';

// Prevent js_scrub from stripping the import
const _dummyFileUploadInputComponent = FileUploadInputComponent;

describe('FileUploadInputComponent', () => {
  let component: FileUploadInputComponent;
  let fixture: ComponentFixture<FileUploadInputComponent>;

  beforeEach(async () => {
    await TestBed
        .configureTestingModule({
          imports: [FileUploadInputComponent],
        })
        .compileComponents();

    fixture = TestBed.createComponent(FileUploadInputComponent);
    component = fixture.componentInstance;
    component.control = new FormControl(null);
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should return accepted file types string', () => {
    component.allowedFileTypes = ['pdf', 'png'];
    expect(component.acceptedFileTypesString).toBe('.pdf, .png');
  });

  it('should handle file change with valid file', () => {
    component.allowedFileTypes = ['pdf'];
    const file = new File([''], 'test.pdf', {type: 'application/pdf'});
    const event = {
      target: {
        files: [file],
      },
    } as any;

    spyOn(component.fileSelected, 'emit');

    component.onFileChange(event);

    expect(component.fileName).toBe('test.pdf');
    expect(component.fileError).toBeNull();
    expect(component.control.value).toBe(file);
    expect(component.fileSelected.emit).toHaveBeenCalledWith(file);
  });

  it('should handle file change with invalid file', () => {
    component.allowedFileTypes = ['pdf'];
    const file = new File([''], 'test.jpg', {type: 'image/jpeg'});
    const event = {
      target: {
        files: [file],
      },
    } as any;

    spyOn(component.fileSelected, 'emit');

    component.onFileChange(event);

    expect(component.fileName).toBeNull();
    expect(component.fileError).toContain('Invalid file type');
    expect(component.control.value).toBeNull();
    expect(component.fileSelected.emit).toHaveBeenCalledWith(null);
  });

  it('should handle file change with no file', () => {
    const event = {
      target: {
        files: [],
      },
    } as any;

    spyOn(component.fileSelected, 'emit');

    component.onFileChange(event);

    expect(component.fileName).toBeNull();
    expect(component.fileError).toBeNull();
    expect(component.control.value).toBeNull();
    expect(component.fileSelected.emit).toHaveBeenCalledWith(null);
  });

  it('should clear file', () => {
    component.fileName = 'test.pdf';
    component.fileError = 'some error';
    component.control.setValue('some value');

    spyOn(component.fileSelected, 'emit');

    component.clearFile();

    expect(component.fileName).toBeNull();
    expect(component.fileError).toBeNull();
    expect(component.control.value).toBeNull();
    expect(component.fileSelected.emit).toHaveBeenCalledWith(null);
  });
});
