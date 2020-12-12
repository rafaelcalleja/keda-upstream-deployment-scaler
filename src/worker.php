<?php

use Externalscaler\ExternalScalerInterface;
use Spiral\Goridge;
use Spiral\RoadRunner;

ini_set('display_errors', 'stderr');
require "vendor/autoload.php";
require __DIR__ . '/ExternalScaler.php';

//To run server in debug mode - new \Spiral\GRPC\Server(null, ['debug' => true]);
$server = new \Spiral\GRPC\Server(null, ['debug' => true]);
$server->registerService(ExternalScalerInterface::class, new ExternalScaler());

$w = new RoadRunner\Worker(new Goridge\StreamRelay(STDIN, STDOUT));
$server->serve($w);
