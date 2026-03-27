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

import {removeEmptyValues} from './utils';

describe('removeEmptyValues', () => {
  it('should remove empty values', () => {
    const input = {
      a: 'value1',
      b: '',
      c: null,
      d: undefined,
      e: 'value2',
    };
    const expected = {
      a: 'value1',
      e: 'value2',
    };
    expect(removeEmptyValues(input)).toEqual(expected);
  });

  it('should handle empty object', () => {
    expect(removeEmptyValues({})).toEqual({});
  });

  it('should remove falsy values like 0 and false', () => {
    const input = {
      a: 0,
      b: false,
      c: 'valid',
    };
    const expected = {
      c: 'valid',
    };
    expect(removeEmptyValues(input)).toEqual(expected);
  });
});
