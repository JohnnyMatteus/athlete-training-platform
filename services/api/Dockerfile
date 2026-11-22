# syntax=docker/dockerfile:1
# swoole e sockets não existem na imagem do composer; a imagem final instala os dois.
# API de importação (Hyperf/Swoole). Dependências na etapa de build; a imagem final não tem compilador.
FROM composer:2 AS deps
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-interaction --prefer-dist --optimize-autoloader --ignore-platform-req=ext-swoole --ignore-platform-req=ext-sockets

FROM php:8.4-cli-alpine3.23 AS runtime
# Extensões do Hyperf: swoole (versão fixada), pgsql para PDO, sockets e pcntl para o servidor.
RUN apk add --no-cache libpq libstdc++ brotli-libs \
 && apk add --no-cache --virtual .build-deps $PHPIZE_DEPS postgresql-dev linux-headers brotli-dev openssl-dev \
 && pecl install swoole-6.2.6 \
 && docker-php-ext-enable swoole \
 && docker-php-ext-install pdo_pgsql pcntl sockets \
 && apk del .build-deps \
 && php --ri swoole > /dev/null && php --ri sockets > /dev/null && php --ri pdo_pgsql > /dev/null
WORKDIR /app
COPY --from=deps /app/vendor ./vendor
COPY . .
RUN addgroup -S app && adduser -S -G app app \
 && mkdir -p /app/runtime && chown app:app /app/runtime
USER app
EXPOSE 9501
ENTRYPOINT ["php", "bin/hyperf.php"]
CMD ["start"]
