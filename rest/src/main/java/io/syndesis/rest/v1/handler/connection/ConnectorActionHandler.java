/**
 * Copyright (C) 2016 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package io.syndesis.rest.v1.handler.connection;

import io.syndesis.dao.manager.DataManager;
import io.syndesis.model.Kind;
import io.syndesis.model.ListResult;
import io.syndesis.model.connection.Action;
import io.syndesis.rest.util.PaginationFilter;
import io.syndesis.rest.util.ReflectiveSorter;
import io.syndesis.rest.v1.handler.BaseHandler;
import io.syndesis.rest.v1.operations.Getter;
import io.syndesis.rest.v1.operations.Lister;
import io.syndesis.rest.v1.operations.PaginationOptionsFromQueryParams;
import io.syndesis.rest.v1.operations.SortOptionsFromQueryParams;
import io.syndesis.rest.v1.util.PredicateFilter;
import io.swagger.annotations.Api;

import javax.ws.rs.WebApplicationException;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.UriInfo;

@Api(value = "actions")
public class ConnectorActionHandler extends BaseHandler implements Lister<Action>, Getter<Action> {

    private final String connectorId;

    ConnectorActionHandler(DataManager dataMgr, String connectorId) {
        super(dataMgr);
        this.connectorId = connectorId;
    }

    @Override
    public Kind resourceKind() {
        return Kind.Action;
    }

    @Override
    public Action get(String id) {
        Action result = Getter.super.get(id);
        if (result == null) {
            throw new WebApplicationException(Response.Status.NOT_FOUND);
        }
        if (result.getConnectorId().equals(connectorId)){
            return result;
        }

        throw new WebApplicationException(Response.Status.NOT_FOUND);
    }

    @Override
    public ListResult<Action> list(UriInfo uriInfo) {
        return getDataManager().fetchAll(
            Action.class,
            new PredicateFilter<>((o) -> o.getConnectorId().equals(connectorId)),
            new ReflectiveSorter<>(Action.class, new SortOptionsFromQueryParams(uriInfo)),
            new PaginationFilter<>(new PaginationOptionsFromQueryParams(uriInfo))
        );
    }
}
