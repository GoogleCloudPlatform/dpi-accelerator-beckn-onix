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

import {HttpClient} from '@angular/common/http';
import {getTestBed, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {ApiService} from './api.service';

// Prevent js_scrub from stripping the import
const _dummyApiService = ApiService;

describe('ApiService', () => {
  let service: ApiService;
  let httpClientSpy: jasmine.SpyObj<HttpClient>;

  beforeEach(() => {
    httpClientSpy = jasmine.createSpyObj('HttpClient', ['get', 'post', 'put']);
    TestBed.configureTestingModule({
      providers: [
        ApiService,
        {provide: HttpClient, useValue: httpClientSpy},
      ],
    });
    service = TestBed.inject(ApiService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should call getGcpRegions and return regions', (done) => {
    const mockRegions = ['us-central1', 'us-east1'];
    httpClientSpy.get.and.returnValue(of(mockRegions));

    service.getGcpRegions().subscribe((regions) => {
      expect(regions).toEqual(mockRegions);
      expect(httpClientSpy.get)
          .toHaveBeenCalledWith('http://localhost:8000/regions');
      done();
    });
  });

  it('should call getGcpProjects and return projects', (done) => {
    const mockProjects = [{projectId: 'p1'}, {projectId: 'p2'}];
    const credentials = {key: 'val'};
    httpClientSpy.post.and.returnValue(of(mockProjects));

    service.getGcpProjects(credentials).subscribe((projects) => {
      expect(projects).toEqual(mockProjects);
      expect(httpClientSpy.post)
          .toHaveBeenCalledWith('http://localhost:8000/projects', credentials);
      done();
    });
  });

  it('should call getGcpProjectNames and return names', (done) => {
    const mockNames = ['p1', 'p2'];
    httpClientSpy.get.and.returnValue(of(mockNames));

    service.getGcpProjectNames().subscribe((names) => {
      expect(names).toEqual(mockNames);
      expect(httpClientSpy.get)
          .toHaveBeenCalledWith('http://localhost:8000/projects');
      done();
    });
  });

  it('should call subscribeToNetwork with payload', (done) => {
    const payload = {foo: 'bar'};
    httpClientSpy.post.and.returnValue(of({status: 'success'}));

    service.subscribeToNetwork(payload).subscribe((res) => {
      expect(res).toEqual({status: 'success'});
      expect(httpClientSpy.post)
          .toHaveBeenCalledWith(
              'http://localhost:8000/api/dynamic-proxy', payload);
      done();
    });
  });

  it('should call postConfigs with payload', (done) => {
    const payload = {config: 'val'};
    httpClientSpy.post.and.returnValue(of({status: 'saved'}));

    service.postConfigs(payload).subscribe((res) => {
      expect(res).toEqual({status: 'saved'});
      expect(httpClientSpy.post)
          .toHaveBeenCalledWith('http://localhost:8000/api/configs', payload);
      done();
    });
  });

  it('should call getConfigPaths and return paths', (done) => {
    const mockPaths = {files: ['p1', 'p2']};
    httpClientSpy.get.and.returnValue(of(mockPaths));

    service.getConfigPaths().subscribe((paths) => {
      expect(paths).toEqual(mockPaths);
      expect(httpClientSpy.get)
          .toHaveBeenCalledWith('http://localhost:8000/api/configs/path');
      done();
    });
  });

  it('should call getConfigData with path and return data', (done) => {
    const mockData = {data: 'val'};
    httpClientSpy.get.and.returnValue(of(mockData));

    service.getConfigData('some/path').subscribe((data) => {
      expect(data).toEqual(mockData);
      expect(httpClientSpy.get)
          .toHaveBeenCalledWith(
              'http://localhost:8000/api/config/data',
              {params: {path: 'some/path'}});
      done();
    });
  });

  it('should call updateConfigData with payload', (done) => {
    const payload = {data: 'new-val'};
    httpClientSpy.put.and.returnValue(of({status: 'updated'}));

    service.updateConfigData(payload).subscribe((res) => {
      expect(res).toEqual({status: 'updated'});
      expect(httpClientSpy.put)
          .toHaveBeenCalledWith(
              'http://localhost:8000/api/config/data', payload);
      done();
    });
  });

  it('should call getState and return state', (done) => {
    const mockState = {state: 'val'};
    httpClientSpy.get.and.returnValue(of(mockState));

    service.getState().subscribe((state) => {
      expect(state).toEqual(mockState);
      expect(httpClientSpy.get)
          .toHaveBeenCalledWith('http://localhost:8000/store');
      done();
    });
  });

  it('should call getInstallerState and return state', (done) => {
    const mockState = {state: 'val'};
    httpClientSpy.get.and.returnValue(of(mockState));

    service.getInstallerState().subscribe((state) => {
      expect(state).toEqual(mockState);
      expect(httpClientSpy.get)
          .toHaveBeenCalledWith('http://localhost:8000/api/installer-state');
      done();
    });
  });

  it('should call storeState with key and value', (done) => {
    httpClientSpy.post.and.returnValue(of({status: 'success'}));

    service.storeState('key1', 'val1').subscribe((res) => {
      expect(res).toEqual({status: 'success'});
      expect(httpClientSpy.post)
          .toHaveBeenCalledWith(
              'http://localhost:8000/store', {key: 'key1', value: 'val1'});
      done();
    });
  });

  it('should call storeBulkState with items', (done) => {
    const items = {key1: 'val1', key2: 'val2'};
    httpClientSpy.post.and.returnValue(of({status: 'success'}));

    service.storeBulkState(items).subscribe((res) => {
      expect(res).toEqual({status: 'success'});
      expect(httpClientSpy.post)
          .toHaveBeenCalledWith('http://localhost:8000/store/bulk', items);
      done();
    });
  });
});
