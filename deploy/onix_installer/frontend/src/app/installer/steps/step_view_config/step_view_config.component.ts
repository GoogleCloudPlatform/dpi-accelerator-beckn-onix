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

import {CommonModule} from '@angular/common';
import {ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit} from '@angular/core';
import {FormsModule} from '@angular/forms';
import {MatButton} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatSpinner} from '@angular/material/progress-spinner';
import {MatTooltip} from '@angular/material/tooltip';
import {Router} from '@angular/router';
import * as yaml from 'js-yaml';

import {ApiService} from '../../../core/services/api.service';
import {InstallerStateService} from '../../../core/services/installer-state.service';
import {removeEmptyValues} from '../../../shared/utils';



interface ConfigFileItem {
  path: string;
  name: string;
  type: 'file'|'folder';
  parentFolder: string|null;
  isHidden: boolean;
  isExpanded: boolean;
}

@Component({
  selector: 'app-step-view-config',
  standalone: true,
  imports:
      [CommonModule, MatButton, MatIcon, MatTooltip, MatSpinner, FormsModule],
  templateUrl: './step_view_config.component.html',
  styleUrl: './step_view_config.component.css',
  changeDetection: ChangeDetectionStrategy.Eager,
})
export class StepViewConfigComponent implements OnInit {
  validationError: string|null = null;
  isLoading = false;

  isEditing = false;
  currentFile: any = null;
  fileContent: string = '';
  isSaving = false;

  // Data
  files: ConfigFileItem[] = [];

  originalFileContent: string = '';

  // editingFile: { path: string, content: string } | null = null;

  constructor(
      private router: Router, private apiService: ApiService,
      private installerService: InstallerStateService,
      private cdr: ChangeDetectorRef) {}

  ngOnInit(): void {
    this.fetchFilePaths();
  }



  fetchFilePaths() {
    this.apiService.getConfigPaths().subscribe(res => {
          const processedFiles: ConfigFileItem[] = [];

          res.files.forEach(path => {
            const parts = path.split('/');

            // If path is "Beckn/source.yaml", parts[0] is the folder
            if (parts.length > 1) {
              const folderName = parts[0];
              // Add folder if not already added
              if (!processedFiles.find(f => f.name === folderName)) {
                processedFiles.push({
                  path: folderName,
                  name: folderName,
                  type: 'folder',
                  parentFolder: null,
                  isHidden: false,
                  isExpanded: false
                });
              }
              // Add the file as a child
              processedFiles.push({
                path,
                name: parts[1],
                type: 'file',
                parentFolder: folderName,
                isHidden: true,
                isExpanded: false
              });
            } else {
              // Root level file
              processedFiles.push({
                path,
                name: path,
                type: 'file',
                parentFolder: null,
                isHidden: false,
                isExpanded: false
              });
            }
          });
          this.files = processedFiles;
          this.cdr.detectChanges();
        });
  }

  onProceedToDeploy() {
    this.router.navigate(['installer', 'view-deployment']);
  }

  toggleFolder(folder: any) {
    folder.isExpanded = !folder.isExpanded;
    // Toggle visibility for all files that belong to this folder
    this.files.forEach(f => {
      if (f.parentFolder === folder.name) {
        f.isHidden = !folder.isExpanded;
      }
    });
  }

  onEditFile(file: any) {
    this.currentFile = file;
    this.isLoading = true;
    this.apiService.getConfigData(file.path).subscribe({
      next: (res) => {
        this.fileContent = res.content;

        this.originalFileContent = res.content;
        this.isEditing = true;
        this.isLoading = false;
        this.cdr.detectChanges();
      },
      error: () => {
        this.isLoading = false;

        // Add this line here as well just in case of an error
        this.cdr.detectChanges();
      }
    });
  }

  private findMissingValue(data: any, prefix: string = ''): string|null {
    // 1. If the value itself is null, we found an empty field
    if (data === null) {
      return prefix;
    }

    // 2. If it's an object, check its children
    if (typeof data === 'object' && data !== undefined) {
      for (const key in data) {
        if (Object.prototype.hasOwnProperty.call(data, key)) {
          const currentPath = prefix ? `${prefix}.${key}` : key;

          // Recursive call
          const missingKey = this.findMissingValue(data[key], currentPath);
          if (missingKey) {
            return missingKey;
          }
        }
      }
    }

    // 3. No missing values found
    return null;
  }

  onSave() {
    // 1. Clear previous errors
    this.validationError = null;
    let parsedDocuments: any[] = [];

    // 2. Client-Side Validation
    try {
      // Use loadAll to support multi-document YAML files (separated by "---")
      yaml.loadAll(this.fileContent, (doc) => {
        parsedDocuments.push(doc);
      });
    } catch (e: any) {
      // 3. Handle Invalid YAML
      console.error('YAML Validation Error:', e);
      this.validationError = `Invalid YAML: ${e.message}`;
      this.cdr
          .detectChanges();  // <--- ADDED: Triggers UI to show the YAML error
      return;                // Stop execution
    }

    // 4. Check for missing values across all YAML documents in the file
    for (const doc of parsedDocuments) {
      const missingField = this.findMissingValue(doc);
      if (missingField) {
        this.validationError = `Configuration Error: The value for '${
            missingField}' cannot be empty.`;
        this.cdr.detectChanges();  // <--- ADDED: Triggers UI to show missing
                                   // field error
        return;                    // Stop execution
      }
    }

    // 5. Check for changes before saving
    if (this.fileContent !== this.originalFileContent) {
      console.log('File content changed. Marking state as modified.');
      this.installerService.updateState({isConfigChanged: true});
    }

    // 6. Proceed to Save
    this.isSaving = true;
    this.cdr.detectChanges();  // <--- ADDED: Triggers UI to show "Saving..."
                               // spinner on button

    const payload = {path: this.currentFile.path, content: this.fileContent};

    this.apiService.updateConfigData(payload).subscribe({
      next: () => {
        this.isSaving = false;
        this.isEditing = false;
        this.cdr.detectChanges();
      },
      error: (err) => {
        this.isSaving = false;
        this.validationError = 'Failed to save changes. Please try again.';
        this.cdr
            .detectChanges();  // <--- ADDED: Show API error if backend fails
      }
    });
  }

  onCancel() {
    this.isEditing = false;
    this.currentFile = null;
    this.fileContent = '';
  }

  onBack() {
    this.router.navigate(['installer', 'app-config'])
  }
}
