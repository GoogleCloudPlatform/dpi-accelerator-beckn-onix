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

import { Component, Input, Output, EventEmitter, OnInit, OnDestroy, Self, Optional } from '@angular/core'; // Added Optional
import { CommonModule } from '@angular/common';
import { AbstractControl, ControlValueAccessor, FormControl, NgControl, ReactiveFormsModule, ValidatorFn, ValidationErrors } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-folder-upload-input',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    ReactiveFormsModule,
  ],
  templateUrl: './folder-upload-input.component.html',
  styleUrls: ['./folder-upload-input.component.css']
})
export class FolderUploadInputComponent implements ControlValueAccessor, OnInit, OnDestroy {
  @Input() label: string = 'Choose Folder';
  @Input() infoText: string = '';
  @Input() control!: FormControl;


  private _actualFormControl!: FormControl;


  private onChange: (value: File[] | null) => void = () => {};
  private onTouched: () => void = () => {};
  private unsubscribe$ = new Subject<void>();

  selectedFolderName: string = 'No folder chosen';
  selectedFilesCount: number = 0;

   constructor(@Optional() @Self() public ngControl: NgControl) {
    if (this.ngControl) {
      this.ngControl.valueAccessor = this;
    }
  }

  ngOnInit(): void {
    if (this.control) {
      this._actualFormControl = this.control;
    } else if (this.ngControl && this.ngControl.control) {
      this._actualFormControl = this.ngControl.control as FormControl;
    } else {
      console.error('FolderUploadInputComponent: No FormControl provided via [control] Input or formControlName.');
      this._actualFormControl = new FormControl(null);
    }

    this._actualFormControl.valueChanges.pipe(takeUntil(this.unsubscribe$)).subscribe(value => {
      if (value instanceof FileList || (Array.isArray(value) && value.every(f => f instanceof File))) {
        const files = Array.from(value);
        if (files.length > 0) {
          const firstFilePath = files[0].webkitRelativePath || files[0].name;
          const folderNameMatch = firstFilePath.match(/^([^/]+)\//);
          this.selectedFolderName = folderNameMatch ? folderNameMatch[1] : 'Selected folder';
          this.selectedFilesCount = files.length;
        } else {
          this.selectedFolderName = 'No folder chosen';
          this.selectedFilesCount = 0;
        }
      } else {
        this.selectedFolderName = 'No folder chosen';
        this.selectedFilesCount = 0;
      }
    });
  }

  ngOnDestroy(): void {
    this.unsubscribe$.next();
    this.unsubscribe$.complete();
  }

  writeValue(value: File[] | null): void {
    if (this._actualFormControl) {
      if (value instanceof FileList) {
        this._actualFormControl.setValue(Array.from(value), { emitEvent: false });
      } else if (Array.isArray(value) && value.every(f => f instanceof File)) {
        this._actualFormControl.setValue(value, { emitEvent: false });
      } else {
        this._actualFormControl.setValue(null, { emitEvent: false });
      }
    }
  }

  registerOnChange(fn: any): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    if (this._actualFormControl) {
      if (isDisabled) {
        this._actualFormControl.disable({ emitEvent: false });
      } else {
        this._actualFormControl.enable({ emitEvent: false });
      }
    }
  }


  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      const files = Array.from(input.files);


      if (files.length > 0) {
        this._actualFormControl.setValue(files);
        this.onChange(files);
      } else {
        this._actualFormControl.setValue(null);
        this.onChange(null);
      }
    } else {
      this._actualFormControl.setValue(null);
      this.onChange(null);
    }
    this.onTouched();
    input.value = '';
  }

  clearSelection(): void {
    if (this._actualFormControl) {
      this._actualFormControl.setValue(null);
      this.onChange(null);
      this.onTouched();
    }
    this.selectedFolderName = 'No folder chosen';
    this.selectedFilesCount = 0;
  }

  get hasError(): boolean {
    return this._actualFormControl?.invalid && (this._actualFormControl?.touched || this._actualFormControl?.dirty);
  }
}