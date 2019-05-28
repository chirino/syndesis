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
package io.syndesis.server.jsondb;

import java.io.InputStream;
import java.io.OutputStream;
import java.util.Set;
import java.util.function.Consumer;

import io.syndesis.common.util.tenant.TenantContext;
import io.syndesis.server.jsondb.impl.Strings;

public class TenantJsonDB implements JsonDB {

    private final JsonDB jsonDB;
    private final TenantContext tenantContext;

    public TenantJsonDB(JsonDB jsonDB, TenantContext tenantContext) {
        this.jsonDB = jsonDB;
        this.tenantContext = tenantContext;
    }

    private String toTenantPath(String path) {
        String tenant = tenantContext.get();
        if ( tenant == null ) {
            throw new JsonDBException("TenantContext not set");
        }
        if( tenant.length()==0 ) {
            return path;
        } else {
            return "/" + tenant + Strings.prefix(path, "/");
        }
    }

    @Override
    public boolean delete(String path) {
        return jsonDB.delete(toTenantPath(path));
    }

    @Override
    public boolean exists(String path) {
        return jsonDB.exists(toTenantPath(path));
    }

    @Override
    public Set<String> fetchIdsByPropertyValue(String path, String property, String value) {
        return jsonDB.fetchIdsByPropertyValue(toTenantPath(path), property, value);
    }

    @Override
    public String createKey() {
        return jsonDB.createKey();
    }

    @Override
    public String getAsString(String path) {
        return jsonDB.getAsString(toTenantPath(path));
    }

    @Override
    public String getAsString(String path, GetOptions options) {
        return jsonDB.getAsString(toTenantPath(path), options);
    }

    @Override
    public void set(String path, String json) {
        jsonDB.set(toTenantPath(path), json);
    }

    @Override
    public void update(String path, String json) {
        jsonDB.update(toTenantPath(path), json);
    }

    @Override
    public String push(String path, String json) {
        return jsonDB.push(toTenantPath(path), json);
    }

    @Override
    public byte[] getAsByteArray(String path) {
        return jsonDB.getAsByteArray(toTenantPath(path));
    }

    @Override
    public byte[] getAsByteArray(String path, GetOptions options) {
        return jsonDB.getAsByteArray(toTenantPath(path), options);
    }

    @Override
    public void set(String path, byte[] json) {
        jsonDB.set(toTenantPath(path), json);
    }

    @Override
    public void update(String path, byte[] json) {
        jsonDB.update(toTenantPath(path), json);
    }

    @Override
    public String push(String path, byte[] json) {
        return jsonDB.push(toTenantPath(path), json);
    }

    @Override
    public void getAsStream(String path, OutputStream os) {
        jsonDB.getAsStream(toTenantPath(path), os);
    }

    @Override
    public boolean getAsStream(String path, GetOptions options, OutputStream os) {
        return jsonDB.getAsStream(toTenantPath(path), options, os);
    }

    @Override
    public Consumer<OutputStream> getAsStreamingOutput(String path) {
        return jsonDB.getAsStreamingOutput(toTenantPath(path));
    }

    @Override
    public Consumer<OutputStream> getAsStreamingOutput(String path, GetOptions options) {
        return jsonDB.getAsStreamingOutput(toTenantPath(path), options);
    }

    @Override
    public void set(String path, InputStream body) {
        jsonDB.set(toTenantPath(path), body);
    }

    @Override
    public void update(String path, InputStream body) {
        jsonDB.update(toTenantPath(path), body);
    }

    @Override
    public String push(String path, InputStream body) {
        return jsonDB.push(toTenantPath(path), body);
    }
}
