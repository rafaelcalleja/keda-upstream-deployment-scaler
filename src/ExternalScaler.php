<?php

use Externalscaler\MetricValue;
use Spiral\GRPC\ContextInterface;
use Externalscaler\ExternalScalerInterface;
use Externalscaler\MetricSpec;
use Externalscaler\ScaledObjectRef;
use Externalscaler\GetMetricsRequest;
use Externalscaler\IsActiveResponse;
use Externalscaler\GetMetricSpecResponse;
use Externalscaler\GetMetricsResponse;

class ExternalScaler implements ExternalScalerInterface
{
    public function IsActive(ContextInterface $ctx, ScaledObjectRef $in): IsActiveResponse
    {
        $isActiveResponse = new IsActiveResponse();
        $namespace = $in->getScalerMetadata()["upstreamDeploymentNamespace"] ?? 'default';
        if (false === isset($in->getScalerMetadata()["upstreamDeploymentName"])) {
            $isActiveResponse->setResult(false);
            return $isActiveResponse;
        }

        $deploymentName = $in->getScalerMetadata()["upstreamDeploymentName"];

        $command = "kubectl --namespace {$namespace} get deployment/$deploymentName -o=jsonpath='{.status.availableReplicas}'";
        $isActive = (bool) is_numeric(exec($command));
        return $isActiveResponse->setResult($isActive);
    }

    public function StreamIsActive(ContextInterface $ctx, ScaledObjectRef $in): IsActiveResponse
    {
        return (new IsActiveResponse())->setResult((bool) $in->getScalerMetadata()['streamActive']);
    }

    public function GetMetricSpec(ContextInterface $ctx, ScaledObjectRef $in): GetMetricSpecResponse
    {
        $metric = new MetricSpec();
        $metric->setMetricName('targetSize');
        $metric->setTargetSize(1);

        return (new GetMetricSpecResponse())->setMetricSpecs([$metric]);
    }

    public function GetMetrics(ContextInterface $ctx, GetMetricsRequest $in): GetMetricsResponse
    {
        $metric = new MetricValue();
        $metric->setMetricName('targetSize');
        $metric->setMetricValue($in->getScaledObjectRef()->getScalerMetadata()['replicaCount'] ?? 1);

        return (new GetMetricsResponse)->setMetricValues([$metric]);
    }
}
