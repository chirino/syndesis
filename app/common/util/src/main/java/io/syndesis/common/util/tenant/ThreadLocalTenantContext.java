/*
 * Copyright (C) 2016 Red Hat, Inc.
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
package io.syndesis.common.util.tenant;

import java.io.Closeable;

final public class ThreadLocalTenantContext implements TenantContext {

    private static final ThreadLocal<String> tenant = new ThreadLocal<>();

    @Override
    public Closeable open(String value) {
        tenant.set(value);
        return () -> {
            tenant.remove();
        };
    }

    @Override
    public String get() {
        return tenant.get();
    }

}
