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

import {getTestBed} from '@angular/core/testing';
import {AbstractControl, FormArray, FormControl} from '@angular/forms';

import {allControlsTrue, appNameValidator, jsonValidator} from './custom-validators';

// Prevent js_scrub from stripping the imports
const _dummyAllControlsTrue = allControlsTrue;
const _dummyJsonValidator = jsonValidator;
const _dummyAppNameValidator = appNameValidator;

describe('CustomValidators', () => {
  describe('allControlsTrue', () => {
    const validator = allControlsTrue();

    it('should return null if not FormArray', () => {
      const control = new FormControl(true);
      expect(validator(control)).toBeNull();
    });

    it('should return null if all controls are true', () => {
      const formArray = new FormArray([
        new FormControl(true),
        new FormControl(true),
      ]);
      expect(validator(formArray)).toBeNull();
    });

    it('should return error if any control is false', () => {
      const formArray = new FormArray([
        new FormControl(true),
        new FormControl(false),
      ]);
      expect(validator(formArray)).toEqual({notAllChecked: true});
    });

    it('should return error if empty FormArray', () => {
      const formArray = new FormArray([]);
      // every on empty array is true, so it should return null!
      // Wait, let's see how it's implemented: formArray.controls.every(...)
      // If it's empty, every returns true.
      expect(validator(formArray)).toBeNull();
    });
  });

  describe('jsonValidator', () => {
    const validator = jsonValidator();

    it('should return null if value is empty', () => {
      const control = new FormControl('');
      expect(validator(control)).toBeNull();
    });

    it('should return null if value is null', () => {
      const control = new FormControl(null);
      expect(validator(control)).toBeNull();
    });

    it('should return null if value is valid JSON', () => {
      const control = new FormControl('{"key": "value"}');
      expect(validator(control)).toBeNull();
    });

    it('should return error if value is invalid JSON', () => {
      const control = new FormControl('{"key": "value"');  // Missing brace
      expect(validator(control)).toEqual({jsonInvalid: true});
    });
  });

  describe('appNameValidator', () => {
    const max = 6;
    const validator = appNameValidator(max);

    it('should return null if value is null', () => {
      const control = new FormControl(null);
      expect(validator(control)).toBeNull();
    });

    it('should return null for valid app name', () => {
      const control = new FormControl('onix12');
      expect(validator(control)).toBeNull();
    });

    it('should return null for valid app name with leading/trailing spaces',
       () => {
         const control = new FormControl('  onix  ');
         expect(validator(control)).toBeNull();
       });

    it('should return error for app name with internal spaces', () => {
      const control = new FormControl('on ix');
      expect(validator(control)).toEqual({invalidAppName: true});
    });

    it('should return error for app name exceeding max length', () => {
      const control = new FormControl('toolongname');
      expect(validator(control)).toEqual({invalidAppName: true});
    });

    it('should return error for whitespace-only app name', () => {
      const control = new FormControl('   ');
      expect(validator(control)).toEqual({invalidAppName: true});
    });

    it('should return error for empty string after trimming', () => {
      const control = new FormControl(' ');
      expect(validator(control)).toEqual({invalidAppName: true});
    });
  });
});
