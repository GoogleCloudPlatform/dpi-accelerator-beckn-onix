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

import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common'; // Needed for *ngIf
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ReactiveFormsModule, FormControl } from '@angular/forms';

@Component({
  selector: 'app-file-upload-input',
  templateUrl: './file-upload-input.component.html',
  styleUrls: ['./file-upload-input.component.css'],
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
  ],
})
export class FileUploadInputComponent {
  @Input() label: string = 'Upload File';
  @Input() control!: FormControl;
  @Input() allowedFileTypes: string[] = [];
  @Input() infoText?: string;

  @Output() fileSelected = new EventEmitter<File | null>();

  fileName: string | null = null;
  fileError: string | null = null;

  // New getter method to compute the accept string
  get acceptedFileTypesString(): string {
    return this.allowedFileTypes.map((type) => `.${type}`).join(', ');
  }

  onFileChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      const file = input.files[0];
      const fileExtension = file.name.split('.').pop()?.toLowerCase();

      if (
        fileExtension &&
        this.allowedFileTypes.includes(fileExtension)
      ) {
        this.fileName = file.name;
        this.fileError = null;
        this.control.setValue(file);
        this.fileSelected.emit(file);
      } else {
        this.fileName = null;
        this.fileError = `Invalid file type. Please upload a file with one of these extensions: ${this.allowedFileTypes.join(', ')}.`;
        this.control.setValue(null);
        this.fileSelected.emit(null);
        input.value = '';
      }
    } else {
      this.fileName = null;
      this.fileError = null;
      this.control.setValue(null);
      this.fileSelected.emit(null);
    }
  }

  clearFile(): void {
    this.fileName = null;
    this.fileError = null;
    this.control.setValue(null);
    this.fileSelected.emit(null);
    const fileInput = document.getElementById('fileUploadInput') as HTMLInputElement;
    if (fileInput) {
      fileInput.value = '';
    }
  }
}