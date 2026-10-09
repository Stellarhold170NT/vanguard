// audit-sample: the second advice (pattern w1-03 pain 5 — ResponseWrapper
// wraps every 2xx beside the error translator; two global envelopes live
// side by side). R5xx-04 single-error-advice is chartered but NOT
// registered in the v0.1 registry, so nothing fires here — the coexistence
// is a declared coverage gap in expectations.json (w5-04 input).
package com.youthunion.audit.config;

import org.springframework.core.MethodParameter;
import org.springframework.http.MediaType;
import org.springframework.http.server.ServerHttpRequest;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.servlet.mvc.method.annotation.ResponseBodyAdvice;

@ControllerAdvice
public class ResponseWrapper implements ResponseBodyAdvice<Object> {

    @Override
    public boolean supports(MethodParameter returnType, Class converterType) {
        return !returnType.hasMethodAnnotation(IgnoreWrapper.class);
    }

    @Override
    public Object beforeBodyWrite(Object body, MethodParameter returnType,
            MediaType selectedContentType, Class selectedConverterType,
            ServerHttpRequest request, ServerHttpResponse response) {
        return body;
    }

    public @interface IgnoreWrapper {
    }
}
