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

import { AbstractControl, FormArray, ValidationErrors, ValidatorFn } from '@angular/forms';

/**
 * Validator that checks if all checkboxes (or controls with boolean value) in a FormArray are true.
 * Returns { notAllChecked: true } if any control's value is not true.
 */
export function allControlsTrue(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control || !(control instanceof FormArray)) {
      return null;
    }

    const formArray = control as FormArray;
    const allTrue =
        formArray.controls.every((c: AbstractControl) => c.value === true);

    return allTrue ? null : { notAllChecked: true };
  };
}


export function jsonValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.value || control.value.trim() === '') {
      return null;
    }

    try {
      JSON.parse(control.value);
    } catch (e) {
      return { jsonInvalid: true };
    }
    return null;
  };
}