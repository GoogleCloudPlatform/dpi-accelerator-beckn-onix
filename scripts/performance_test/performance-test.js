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

import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Get configuration from environment variables with sensible defaults
const APP_BASE_URL = __ENV.APP_BASE_URL || 'http://localhost:8080';
const TARGET_TPS = parseInt(__ENV.TARGET_TPS || '10'); // Default to 10 TPS
const TEST_DURATION = __ENV.TEST_DURATION || '1m'; // Default to 1 minute
const VUS_PER_TPS = parseInt(__ENV.VUS_PER_TPS || '5'); // VUs to pre-allocate per TPS
const TRANSACTIONS_TO_RUN = __ENV.TRANSACTIONS ? __ENV.TRANSACTIONS.split(',') : ['search'];

// Central mapping of transaction names to API endpoints
const transactionEndpoints = {
    search: '/bap/caller/search',
    select: '/bap/caller/select',
    init: '/bap/caller/init',
    confirm: '/bap/caller/confirm',
    status: '/bap/caller/status',
    track: '/bap/caller/track',
    update: '/bap/caller/update',
    cancel: '/bap/caller/cancel',
    rating: '/bap/caller/rating',
    support: '/bap/caller/support',
    on_search: '/bpp/caller/on_search',
    on_select: '/bpp/caller/on_select',
    on_init: '/bpp/caller/on_init',
    on_confirm: '/bpp/caller/on_confirm',
    on_status: '/bpp/caller/on_status',
    on_track: '/bpp/caller/on_track',
    on_update: '/bpp/caller/on_update',
    on_cancel: '/bpp/caller/on_cancel',
    on_rating: '/bpp/caller/on_rating',
    on_support: '/bpp/caller/on_support',
};

// Load payload data dynamically based on the transactions to run
const transactionPayloads = {};
for (const tx of TRANSACTIONS_TO_RUN) {
    let filePath;
    try {
        filePath = `${tx}.json`;
        transactionPayloads[tx] = new SharedArray(`${tx}Data`, function () {
            return JSON.parse(open(filePath));
        });
    } catch (e) {
        throw new Error(`Could not load payload for transaction '${tx}'. Attempted to read from path: '${filePath}' Please ensure the file exists and that the k6 command is run from the correct directory. Original error: ${e}`);
    }
}

// Function to dynamically generate scenarios and thresholds
function setupTestOptions(transactions) {
    const scenarios = {};
    const thresholds = {
        'http_req_failed': ['rate<0.01'], // failure rate should be less than 1%
        'http_req_duration': ['p(95)<100'], // 95th percentile response time should be < 100ms
    };

    for (const tx of transactions) {
        // Create a scenario for each transaction
        scenarios[`${tx}_scenario`] = {
            executor: 'constant-arrival-rate',
            rate: TARGET_TPS,
            timeUnit: '1s',
            duration: TEST_DURATION,
            preAllocatedVUs: Math.ceil(TARGET_TPS * 2),
            maxVUs: TARGET_TPS * VUS_PER_TPS, // Scale maxVUs with load
            exec: 'runTransaction',
            env: { TRANSACTION_NAME: tx }, // Pass transaction name to the exec function
        };

        // Create a specific threshold for each transaction
        thresholds[`http_req_duration{transaction:${tx}}`] = ['p(95)<50']; // 95% of these reqs should be < 50ms
        thresholds[`http_req_failed{transaction:${tx}}`] = ['rate<0.01']; // Failure rate for these reqs < 1%
    }
    return { scenarios, thresholds };
}

const testOptions = setupTestOptions(TRANSACTIONS_TO_RUN);
export const options = {
    scenarios: testOptions.scenarios,
    thresholds: testOptions.thresholds,
};

/**
 * A generic function to execute any Beckn transaction.
 * It reads the TRANSACTION_NAME from the scenario's environment.
 */
export function runTransaction() {
    // Get the specific transaction to run for this VU from the scenario env
    const transactionName = __ENV.TRANSACTION_NAME;

    // Get a random payload from the shared array and the corresponding endpoint
    const payloads = transactionPayloads[transactionName];
    const payloadTemplate = payloads[Math.floor(Math.random() * payloads.length)];
    const endpoint = transactionEndpoints[transactionName];

    if (!payloadTemplate || !endpoint) {
        console.error(`Configuration missing for transaction: ${transactionName}`);
        return;
    }

    const payload = JSON.parse(JSON.stringify(payloadTemplate));

    payload.context.message_id = uuidv4();
    payload.context.transaction_id = uuidv4();
    payload.context.timestamp = new Date().toISOString();

    const headers = { 'Content-Type': 'application/json' };
    const res = http.post(
        `${APP_BASE_URL}${endpoint}`,
        JSON.stringify(payload),
        {
            headers: headers,
            tags: { transaction: transactionName }
        }
    );

    check(res, {
        [`${transactionName} status is 200`]: (r) => r.status === 200,
    }, { transaction: transactionName });
}