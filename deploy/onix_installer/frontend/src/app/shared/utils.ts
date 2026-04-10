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

import { AbstractControl, ValidationErrors, ValidatorFn } from "@angular/forms";

export const removeEmptyValues = (obj: { [key: string]: any }) => {
  const newObj: { [key: string]: any } = {};
  for (const key in obj) {
    if (obj[key]) {
      newObj[key] = obj[key];
    }
  }
  return newObj;
};

export const sanitizeFormValues = <T>(config: T): T => {
  if (config === null || typeof config !== 'object') {
    if (typeof config === 'string') {
      return (config.trim() as T);
    }
    return config;
  }

  // Handle File and Blob objects - return as is
  if (config instanceof File || config instanceof Blob) {
    return config;
  }

  if (Array.isArray(config)) {
    return config.map(item => sanitizeFormValues(item)) as T;
  }

  const sanitizedConfig: {[key: string]: any} = {};
  for (const key of Object.keys(config)) {
    const value = (config as any)[key];
    sanitizedConfig[key] = sanitizeFormValues(value);
  }
  return sanitizedConfig as T;
};
