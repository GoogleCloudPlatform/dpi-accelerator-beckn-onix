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

import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class ApiService {
  private apiUrl = 'http://localhost:8000/';

  constructor(private http: HttpClient) {}

  getGcpRegions(): Observable<string[]> {
    return this.http.get<string[]>(`${this.apiUrl}regions`);
  }
  getGcpProjects(credentials: any): Observable<any[]> {
    return this.http.post<any[]>(`${this.apiUrl}projects`, credentials);
  }

  getGcpProjectNames(): Observable<string[]> {
    return this.http.get<string[]>(`${this.apiUrl}projects`);
  }

  subscribeToNetwork(payload: any): Observable<any> {
    const subscribeUrl = `${this.apiUrl}api/dynamic-proxy`;
    return this.http.post(subscribeUrl, payload);
  }

    getState(): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}store`);
  }


  storeState(key: string, value: any): Observable<any> {
    return this.http.post(`${this.apiUrl}store`, { key, value });
  }

  storeBulkState(items: { [key: string]: any }): Observable<any> {
    return this.http.post(`${this.apiUrl}store/bulk`, items);
  }
}