<?php

declare(strict_types=1);

use Hyperf\Di\Container;
use Hyperf\Di\Definition\DefinitionSourceFactory;
use Hyperf\Support\DotenvManager;

DotenvManager::load([BASE_PATH]);

$container = new Container((new DefinitionSourceFactory())());
Hyperf\Context\ApplicationContext::setContainer($container);

return $container;
