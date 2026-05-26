# Changelog

## [1.8.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.7.1...v1.8.0) (2026-05-26)


### Features

* add GetShopByID method to retrieve shop details by ID ([9f49602](https://github.com/Encargalo/Encargalo.app-api.go/commit/9f496023aa0fc82493a74ddbd6f9925fce61ccd8))
* Enhance API documentation and functionality ([42be9c7](https://github.com/Encargalo/Encargalo.app-api.go/commit/42be9c797718b54c43b9c8409c8443a097e84623))
* Implement user roles management and update authentication logic ([fc1328e](https://github.com/Encargalo/Encargalo.app-api.go/commit/fc1328efc50d03b8182baad2f1537ac19bd4edb5))


### Refactors

* remove employee group and related logic; update user group to include shop information ([2ced36f](https://github.com/Encargalo/Encargalo.app-api.go/commit/2ced36fd16c873f46140c0e2fa512d685679e87d))
* remove unused credit discount logic and related models ([c4481c0](https://github.com/Encargalo/Encargalo.app-api.go/commit/c4481c0e935944d846c66e104e4ace38c0a74441))

## [1.7.1](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.7.0...v1.7.1) (2026-05-19)


### Bug Fixes

* add missing go.sum entries for otel semconv/trace (pgdriver@v1.2.16) ([802fc06](https://github.com/Encargalo/Encargalo.app-api.go/commit/802fc0660cc9cb334201525a2387e74ddae5ca75))

## [1.7.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.6.0...v1.7.0) (2026-05-18)


### Features

* add reference and bolivianos prices to item and addition responses ([053a215](https://github.com/Encargalo/Encargalo.app-api.go/commit/053a215c2b8ee91f73c941142881141af13c99f5))
* add reference and bolivianos prices to item and addition responses ([e458ee8](https://github.com/Encargalo/Encargalo.app-api.go/commit/e458ee8c7f63babfb840c8d02c5054888e0d0f47))

## [1.6.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.5.4...v1.6.0) (2026-05-16)


### Features

* **bcv:** add exchange rate scraper, banking holiday cron, and seed migrations ([bef6a08](https://github.com/Encargalo/Encargalo.app-api.go/commit/bef6a0819c0ca66a0fda241cf399419e7813e22c))
* **bcv:** add exchange rate scraper, banking holiday cron, and seed migrations ([c0b161c](https://github.com/Encargalo/Encargalo.app-api.go/commit/c0b161c0a60cf3b83e1b5f8af0288c3b9e845705))

## [1.5.4](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.5.3...v1.5.4) (2026-03-24)


### Bug Fixes

* **docker:** harden container image against CVE-2025-60876 ([d297884](https://github.com/Encargalo/Encargalo.app-api.go/commit/d297884b35fcf351e56d8f296e51dd034fa6a61a))
* **docker:** harden container image against CVE-2025-60876 ([8a233ba](https://github.com/Encargalo/Encargalo.app-api.go/commit/8a233ba53ceb8f9dc6ca6e43ab02ded6a0d03404))

## [1.5.3](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.5.2...v1.5.3) (2026-03-24)


### Bug Fixes

* **docker:** revert alpine version from 3.23.3 to 3.21.6 ([ded45be](https://github.com/Encargalo/Encargalo.app-api.go/commit/ded45be9a4e2317b103ef0540e4a4ddc7fa21267))
* **docker:** revert alpine version from 3.23.3 to 3.21.6 ([069116a](https://github.com/Encargalo/Encargalo.app-api.go/commit/069116a430440434d71da2c96ec838624392de1b))

## [1.5.2](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.5.1...v1.5.2) (2026-03-24)


### Bug Fixes

* **docker:** bump alpine 3.21.6→3.23.3 and alpine builder 3.22→3.23 ([05754f7](https://github.com/Encargalo/Encargalo.app-api.go/commit/05754f7fd507ad8c198399c8e5b602745e25b13a))
* **docker:** bump alpine 3.21.6→3.23.3 and alpine builder 3.22→3.23 ([ef2c57a](https://github.com/Encargalo/Encargalo.app-api.go/commit/ef2c57ae9988fa6b54ed0d23021ed212951a22e1))

## [1.5.1](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.5.0...v1.5.1) (2026-03-24)


### Bug Fixes

* **orders:** add role-based auth and improve websocket protocol ([05877e6](https://github.com/Encargalo/Encargalo.app-api.go/commit/05877e65fd0ba501c9f7fe42b450bd50f1d17e0f))
* **orders:** add role-based auth and improve websocket protocol ([f1b902e](https://github.com/Encargalo/Encargalo.app-api.go/commit/f1b902ec48e5c7452ace316c96a07bb6b428c4ab))

## [1.5.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.4.0...v1.5.0) (2026-03-24)


### Features

* (shops): se agregó logica condicional para que no se calcule distancia ni tiempo al consultar los negpocios. ([0ea91b1](https://github.com/Encargalo/Encargalo.app-api.go/commit/0ea91b18e676651cc5edacd0af0d13360b3d4209))
* add dockerfile ([bc3a427](https://github.com/Encargalo/Encargalo.app-api.go/commit/bc3a427b5d45ca3bc95185eb4cf991928dc74a25))
* Add new product-related endpoints and data structures ([7e9c746](https://github.com/Encargalo/Encargalo.app-api.go/commit/7e9c7468b2d1787a89a506ca71342f163d3c34ae))
* Add new product-related endpoints and data structures ([378f398](https://github.com/Encargalo/Encargalo.app-api.go/commit/378f39854e381b3fd22bbeb7b3f3550d170da589))
* add peticion post ([b61dd07](https://github.com/Encargalo/Encargalo.app-api.go/commit/b61dd07a28c24a52926f5dd94e8fac638675ffdd))
* add release automation and Docker publish workflows ([543b8ae](https://github.com/Encargalo/Encargalo.app-api.go/commit/543b8aed133c9eaae3919cbc1309c357484cb5ae))
* add release automation and Docker publish workflows ([dc322c6](https://github.com/Encargalo/Encargalo.app-api.go/commit/dc322c6c3a8030882cb36af35506fb3172fe1ee4))
* **address:** se cambio la base de datos de las direcciones de mongodb a postgres. ([c8395b1](https://github.com/Encargalo/Encargalo.app-api.go/commit/c8395b1f40730d0062ddc3e9b049fbf02ee7515e))
* **address:** se eliminió la columna de reference de la base de datos. ([2c4dd43](https://github.com/Encargalo/Encargalo.app-api.go/commit/2c4dd43f7b3d231d655fe31e1523b184c051d072))
* agregar campo OrderID en DataEventOrderCreated y ajustar lógica en BuidlToModelEvent ([a4a1a1a](https://github.com/Encargalo/Encargalo.app-api.go/commit/a4a1a1ad6c3a89d009f0a2ba5a8b2acf283df0f7))
* agregar campo OrderID en DataEventOrderCreated y ajustar lógica en BuidlToModelEvent ([ce3d591](https://github.com/Encargalo/Encargalo.app-api.go/commit/ce3d591df318f3f8c968ee95820bfe97725b784a))
* agregar manejo de cierre de sesión para clientes y empleados, actualizando la lógica de cookies ([f2b2882](https://github.com/Encargalo/Encargalo.app-api.go/commit/f2b2882459cacaf1b280b6a07dc1e4e4cc7dbe74))
* agregar manejo de empleados con nuevas rutas y DTOs ([694f5fd](https://github.com/Encargalo/Encargalo.app-api.go/commit/694f5fdc26eb90210da854917fae94a02eaf8d95))
* agregar manejo de empleados con nuevas rutas y DTOs ([879b45c](https://github.com/Encargalo/Encargalo.app-api.go/commit/879b45c6273aef82e908f3319fc102628c45a5e1))
* agregar manejo del entorno en la configuración y ajustar lógica de consulta en producción ([cf57b94](https://github.com/Encargalo/Encargalo.app-api.go/commit/cf57b944598f67352a531d32a92ff8622cb3b22c))
* agregar nombre y teléfono del cliente en el modelo de orden y en la aplicación de órdenes ([35cc36c](https://github.com/Encargalo/Encargalo.app-api.go/commit/35cc36cce576d8d51dfa9a000869d25b121da2f7))
* agregar nombre y teléfono del cliente en el modelo de orden y en la aplicación de órdenes ([c380308](https://github.com/Encargalo/Encargalo.app-api.go/commit/c380308063481f0795ac3b7a7f7571a906463c1b))
* ajustar lógica de consulta para entornos de producción en GetAllShops y GetShopsBy ([8c7f618](https://github.com/Encargalo/Encargalo.app-api.go/commit/8c7f618ac1993034dca6a83d889eb6deb4e56d03))
* **auth, employee, orders:** agregar autenticación de empleados y gestión de órdenes por WebSocket ([44fc07e](https://github.com/Encargalo/Encargalo.app-api.go/commit/44fc07eb34076109fb51e73a18db1aae794977b9))
* **auth:** add JWT session management and refactor authentication handler ([6182b27](https://github.com/Encargalo/Encargalo.app-api.go/commit/6182b27563eab6dc06ea8a9f8c2d70e5c6e8c011))
* **auth:** add session search functionality and update session management logic ([d537955](https://github.com/Encargalo/Encargalo.app-api.go/commit/d537955aa512cc46ad3e8174594bcc422a3b8f1f))
* **auth:** agregar endpoints para inicio de sesión de clientes y empleados ([8e1d626](https://github.com/Encargalo/Encargalo.app-api.go/commit/8e1d6262feb89b6545554f3343a807184de265aa))
* **auth:** agregar funcionalidad para eliminar sesiones activas y actualizar la documentación de la API ([32bf0a2](https://github.com/Encargalo/Encargalo.app-api.go/commit/32bf0a277c1f2589f86c6a5c2b844bd42e53a5fb))
* **auth:** agregar funcionalidad para eliminar sesiones activas y actualizar la documentación de la API ([bb35754](https://github.com/Encargalo/Encargalo.app-api.go/commit/bb35754653835e6c118620a7f71992acd28a21e8))
* **auth:** enhance authentication middleware with session validation ([d537955](https://github.com/Encargalo/Encargalo.app-api.go/commit/d537955aa512cc46ad3e8174594bcc422a3b8f1f))
* **auth:** implement customer sign-in functionality with JWT support ([6155e8d](https://github.com/Encargalo/Encargalo.app-api.go/commit/6155e8d1e7c987fd2f5be795cf2a149d49ac3596))
* **auth:** integrate Redis for session management and update authentication logic ([21f506a](https://github.com/Encargalo/Encargalo.app-api.go/commit/21f506a4b73f6c86f9aa318cf5f32e0abeade31d))
* **auth:** mejorar manejo de sesiones y roles; actualizar middleware y modelos ([db9bef1](https://github.com/Encargalo/Encargalo.app-api.go/commit/db9bef170a6102078de5134d331109b988bb0dc5))
* **auth:** refactor middleware and session handling; add role management ([fac62c8](https://github.com/Encargalo/Encargalo.app-api.go/commit/fac62c8fe15b230b8e18149b4e8bed477c334d71))
* **auth:** refactor y unificar middleware de autenticación para clientes y empleados ([86de5ad](https://github.com/Encargalo/Encargalo.app-api.go/commit/86de5ad05337d103efcd8c133d0ec6e24a88f563))
* **auth:** unificar manejo de cierre de sesión y actualizar rutas de autenticación ([ca78536](https://github.com/Encargalo/Encargalo.app-api.go/commit/ca78536f198f0cab97159d722e8493510598eac9))
* **CI:** agg archivo de continuos integration. ([729fb54](https://github.com/Encargalo/Encargalo.app-api.go/commit/729fb54c080695a31c71359607b79bb25147c5f3))
* **CI:** agg archivo de continuos integration. ([0916128](https://github.com/Encargalo/Encargalo.app-api.go/commit/0916128f7daf5e340323c51a9894ecd001fb05fc))
* **cookie:** actualizar la política SameSite de las cookies a None p… ([ff3a5d2](https://github.com/Encargalo/Encargalo.app-api.go/commit/ff3a5d2fc657cd87b58451e94daaa7cc939cc1e2))
* **cookie:** actualizar la política SameSite de las cookies a None para permitir el uso en contextos de terceros ([3e6df82](https://github.com/Encargalo/Encargalo.app-api.go/commit/3e6df8237f7a64a56d043ccba43991b611cb4265))
* **cookie:** actualizar la política SameSite de las cookies a Strict para mejorar la seguridad ([49be14f](https://github.com/Encargalo/Encargalo.app-api.go/commit/49be14f912686f886c58a7a9ef238b306dd848df))
* **creditDiscount:** feat(creditDiscount):  ([2a87552](https://github.com/Encargalo/Encargalo.app-api.go/commit/2a8755202c9b57439d02c4cc896bcbb99b9fdc0d))
* **customers:** add customer management endpoints and address handling ([f375f38](https://github.com/Encargalo/Encargalo.app-api.go/commit/f375f38510d18f7ec111c1c12f62ea4fee306c0a))
* **customers:** implement customer group routing and middleware integration ([d537955](https://github.com/Encargalo/Encargalo.app-api.go/commit/d537955aa512cc46ad3e8174594bcc422a3b8f1f))
* **customers:** mejorar la búsqueda de clientes y manejar errores de forma más clara ([da56b5e](https://github.com/Encargalo/Encargalo.app-api.go/commit/da56b5ed9ed59ce900d90672ae32826f732db778))
* **docker:** actualizar la imagen base de Alpine a 3.21 ([34c0101](https://github.com/Encargalo/Encargalo.app-api.go/commit/34c010129813dc3e37cdcd3edb0936abde8973c4))
* **docker:** actualizar la imagen base de Alpine a 3.21 ([d98c42d](https://github.com/Encargalo/Encargalo.app-api.go/commit/d98c42d5f4964a191f3303cdefa5f3d875cabf5a))
* **docker:** actualizar la imagen de Go a 1.25.5 y corregir el puerto expuesto a 3000 ([a5cdcd9](https://github.com/Encargalo/Encargalo.app-api.go/commit/a5cdcd9d0cc601e03c2de5f086d1933374a97460))
* eliminar el grupo de autenticación y ajustar la lógica de cierre de sesión para empleados ([b4fdc52](https://github.com/Encargalo/Encargalo.app-api.go/commit/b4fdc52667fafe1ad99b590900044d4eeec27916))
* **employee:** agregar middleware de autenticación para empleados y métodos de búsqueda por ID ([b872a89](https://github.com/Encargalo/Encargalo.app-api.go/commit/b872a89bf7dca4e957bb6731e0a80df352db83fe))
* enhance products domain with flavors, image update, and orders image snapshot ([11c40cd](https://github.com/Encargalo/Encargalo.app-api.go/commit/11c40cd545a7be556270cabc08eb937a5f197b57))
* enhance products domain with flavors, image update, and orders image snapshot ([f645413](https://github.com/Encargalo/Encargalo.app-api.go/commit/f64541388e55cef29d4a18d15c10b5a01c77aa2b))
* **go:** actualizar la versión de Go a 1.25 y actualizar dependencias ([6ccfc8f](https://github.com/Encargalo/Encargalo.app-api.go/commit/6ccfc8fc85abf8e8271d803869fec0ab82a3c5e5))
* implement shops API with handlers, repository, and Swagger documentation ([c01a8f6](https://github.com/Encargalo/Encargalo.app-api.go/commit/c01a8f62437e76ed9dfd04b56f6e94ad21061597))
* implementar lógica dinámica para el atributo SameSite en las cookies. ([30052e2](https://github.com/Encargalo/Encargalo.app-api.go/commit/30052e28dd8f629f757d6700c98dd75a8060f96c))
* initialize Go project with basic health check and Swagger documentation ([7593bf9](https://github.com/Encargalo/Encargalo.app-api.go/commit/7593bf9ee264b96896976a01aef3ffe9c9c70a78))
* **logging:** integrate Slack logging for error handling and notifications ([6aa2c24](https://github.com/Encargalo/Encargalo.app-api.go/commit/6aa2c24637c27a35f6ff3f84928ff15208930bca))
* **migrations:** agregar columnas customer_name y phone_customer a la tabla de órdenes ([89b2fe0](https://github.com/Encargalo/Encargalo.app-api.go/commit/89b2fe0d37aa8d4d605aa1362c59103bc44db2ad))
* **order:** agregar campo 'number' a la entidad Order y actualizar la lógica de creación de pedidos ([c360166](https://github.com/Encargalo/Encargalo.app-api.go/commit/c360166fe4475f371323fe30eeabcbe8f44b527c))
* **order:** agregar campo 'number' a la entidad Order y actualizar la lógica de creación de pedidos ([ebabb78](https://github.com/Encargalo/Encargalo.app-api.go/commit/ebabb78963dd04fdfa3fbd582c014904a919b402))
* **order:** agregar campo de tarifa de entrega en la creación de pedidos ([58b58b0](https://github.com/Encargalo/Encargalo.app-api.go/commit/58b58b03cfd65f56303a93cc52ec9382549649a7))
* **order:** eliminar el campo 'name' de CreateAdditionsOrders y actualizar la lógica de asignación de nombres en la creación de pedidos ([13cda25](https://github.com/Encargalo/Encargalo.app-api.go/commit/13cda2572ad239aed7e362da2f19f79e43db7d06))
* **order:** eliminar el campo 'name' de CreateItemsOrder y actualizar la lógica de asignación de nombres en la creación de pedidos ([41fd2a6](https://github.com/Encargalo/Encargalo.app-api.go/commit/41fd2a605933306e702beae705a3e423038b3009))
* **order:** incluir relaciones de ItemsOrders y Additions en la búsqueda de pedidos por ID ([b127fd1](https://github.com/Encargalo/Encargalo.app-api.go/commit/b127fd1e5dcea9ef6ab30d61122f1c0f005263aa))
* **order:** refactor UpdateOrderStatus to use typed constants for order statuses and payment methods ([118d7aa](https://github.com/Encargalo/Encargalo.app-api.go/commit/118d7aa03f53ddb14e0ef3a89ff059bbacf59981))
* **order:** refactor UpdateOrderStatus to use typed constants for order statuses and payment methods ([2c35dcc](https://github.com/Encargalo/Encargalo.app-api.go/commit/2c35dcc37c5615932b3a283825608f48ee96c16e))
* **orders, shops:** agregar búsqueda de batch por ID de tienda y act… ([f13207a](https://github.com/Encargalo/Encargalo.app-api.go/commit/f13207a679c04f8df0629218ce3f3edc8743e1d7))
* **orders, shops:** agregar búsqueda de batch por ID de tienda y actualizar modelo de orden ([73da058](https://github.com/Encargalo/Encargalo.app-api.go/commit/73da058f328ca7e8839f93b06b4daf7b943b25ec))
* **orders:** add order management functionality with creation endpoint and data models ([a41031a](https://github.com/Encargalo/Encargalo.app-api.go/commit/a41031af8ba441b87bd3f1008091ccb1739bc536))
* **orders:** add order management functionality with creation endpoint and data models ([5a829c9](https://github.com/Encargalo/Encargalo.app-api.go/commit/5a829c9dcd3754a4ea3a65e9fd474e1d346a95c8))
* **orders:** agregar actualización de estado de órdenes y soporte para múltiples conexiones WebSocket ([5902727](https://github.com/Encargalo/Encargalo.app-api.go/commit/5902727a7d8df44fafdad5d0d3e96188a9d25ba6))
* **orders:** agregar búsqueda de órdenes abiertas por batch y mejorar gestión de conexiones WebSocket ([cbcaf62](https://github.com/Encargalo/Encargalo.app-api.go/commit/cbcaf62a4ff79b29e0f879c6ce08024d94b10465))
* **orders:** agregar evento para la creación de órdenes en el flujo de Redis ([cf8cc8e](https://github.com/Encargalo/Encargalo.app-api.go/commit/cf8cc8eb9af0a152338361059170b5885f88c573))
* **orders:** agregar evento para la creación de órdenes en el flujo de Redis ([b71e531](https://github.com/Encargalo/Encargalo.app-api.go/commit/b71e5311ed5651b21791dc620e71537993fedb30))
* **orders:** agregar manejo de fecha de cancelación al actualizar el estado de la orden ([ae74dcf](https://github.com/Encargalo/Encargalo.app-api.go/commit/ae74dcfe3f885ebb3071c3831a1c0490d2dd4419))
* **orders:** agregar relaciones de ItemsOrders y Additions en la búsqueda de órdenes por batch abierto ([879c76a](https://github.com/Encargalo/Encargalo.app-api.go/commit/879c76a6e9f7625edc8eb566024587a5c01d99b4))
* **orders:** agregar soporte para WebSocket en la gestión de órdenes y actualizar la estructura de datos ([36be452](https://github.com/Encargalo/Encargalo.app-api.go/commit/36be4523283b6fb73abe04dfd28884e94a1ed501))
* **orders:** implement Redis stream integration for order processing ([4d4f447](https://github.com/Encargalo/Encargalo.app-api.go/commit/4d4f44732222dc8f2e2b84aa9f405afcd86229d0))
* **orders:** integrate Redis stream for order processing and add producer functionality ([94e7b5d](https://github.com/Encargalo/Encargalo.app-api.go/commit/94e7b5d9b7ecc1cfa5ccdabe594f1be9bf114d1c))
* **orders:** integrate Redis stream for order processing and add producer functionality ([b8c5843](https://github.com/Encargalo/Encargalo.app-api.go/commit/b8c58434d0fddc56310a12a42bfbfe984c1e5602))
* **orders:** mejorar gestión de errores en la búsqueda de órdenes por batch abierto y optimizar el manejo de conexiones WebSocket ([b63f7c3](https://github.com/Encargalo/Encargalo.app-api.go/commit/b63f7c30d14a59c6f2edf84fa4cb8e0730471084))
* **orders:** se agg lógica para implementar el campo details en la base de datos. ([0a9e730](https://github.com/Encargalo/Encargalo.app-api.go/commit/0a9e730075210975df79338c5fb309e87f72fc7c))
* **orders:** se agg peticion http para envio de peticion para descontar el % de los creditos de los negocios. ([d4ce708](https://github.com/Encargalo/Encargalo.app-api.go/commit/d4ce708351727968e8b0ff872d7ec5443c8f729d))
* **orders:** se liminaron los campos de customer_nae y phone_customer de las ordenes, también se agrego la migración correspondiente. ([24bf03e](https://github.com/Encargalo/Encargalo.app-api.go/commit/24bf03e62bfb3731a49b2758704a6fd5ec27bf02))
* **postgres:** add migrations system ([dc009bb](https://github.com/Encargalo/Encargalo.app-api.go/commit/dc009bba5aa8ebaf57d6d1e02131319159d87c04))
* **products:** actualizar manejo de imágenes en el registro de productos; incluir ID y hash de imagen ([5cc8766](https://github.com/Encargalo/Encargalo.app-api.go/commit/5cc87664393951efc4dc11066874046b7f81b07c))
* **products:** add pagination to all product listing endpoints ([d18649c](https://github.com/Encargalo/Encargalo.app-api.go/commit/d18649c7e45569725266b235eee2fbcae8f0c390))
* **products:** add pagination to all product listing endpoints ([754d66a](https://github.com/Encargalo/Encargalo.app-api.go/commit/754d66a4e73b8f15709d510acb55957eff1799bc))
* **products:** corregir manejo de ID público de imagen y actualizar código de estado de respuesta al registrar productos ([2ff2977](https://github.com/Encargalo/Encargalo.app-api.go/commit/2ff29770ee1bb0f6fc112b199e759acd21282b3d))
* **products:** implementar actualización de productos con validación y manejo de errores ([e77b348](https://github.com/Encargalo/Encargalo.app-api.go/commit/e77b348cad36e10f2370c4915c9ed1eda5be0205))
* **products:** implementar registro de productos con carga de imagen y validación ([ab13928](https://github.com/Encargalo/Encargalo.app-api.go/commit/ab1392887b5c2d9e5fe79eec7368ddbb511b950c))
* **products:** mejorar manejo de errores al registrar productos; agregar errores específicos para carga y conversión de imágenes ([bb9fefa](https://github.com/Encargalo/Encargalo.app-api.go/commit/bb9fefa5c3e706cef97618d73f6d824baa3f9b73))
* **products:** se agregó un cashe de una hora para los productos de cada negocio ([09238ce](https://github.com/Encargalo/Encargalo.app-api.go/commit/09238cebfa2f05e4feff1148b8f2a2e16aa0a9c8))
* **products:** se agregó un cashe de una hora para los productos de cada negocio ([ea0f71d](https://github.com/Encargalo/Encargalo.app-api.go/commit/ea0f71d168733b5a6419e754cb3963347fad4e6d))
* **products:** se mejoraron las respuesta de los errores en el handler ([c58c64b](https://github.com/Encargalo/Encargalo.app-api.go/commit/c58c64bdc51a43350d55bd040f914e75bbdd69f0))
* refactor manejo de cookies y sesiones, eliminando dependencias innecesarias y mejorando la lógica de creación y eliminación de cookies ([e50b7d8](https://github.com/Encargalo/Encargalo.app-api.go/commit/e50b7d8df5b07cedd22494b1995c8c2c97e87239))
* **release:** add Docker publish workflow after release creation ([e90899b](https://github.com/Encargalo/Encargalo.app-api.go/commit/e90899b31e991eb436d4085125ace832be96029c))
* **release:** add Docker publish workflow after release creation ([34fe886](https://github.com/Encargalo/Encargalo.app-api.go/commit/34fe886017698bdbbdc993fc79b69eebe1ad27f7))
* reorganizar grupos de rutas para pedidos de clientes y empleados ([6e9202c](https://github.com/Encargalo/Encargalo.app-api.go/commit/6e9202c19d9b062db9feebf0f8095d07189f7d9d))
* reorganizar grupos de rutas para pedidos de clientes y empleados ([5b9fe01](https://github.com/Encargalo/Encargalo.app-api.go/commit/5b9fe0155c17622d891f9bda30bce7c83f944051))
* se agg logica para inciar sesión cuando se registra un nuevo customer ([9956f9d](https://github.com/Encargalo/Encargalo.app-api.go/commit/9956f9dd7f680ec27e28e795f764e97ee6c78e0c))
* se agg logica para inciar sesión cuando se registra un nuevo customer ([50407d0](https://github.com/Encargalo/Encargalo.app-api.go/commit/50407d01b3800ce0f45e39a574c70e3253b7209e))
* **shops:** agregar información de distancia y tiempo a la respuesta de tiendas, incluyendo cálculo de tarifas de entrega ([ef9bf74](https://github.com/Encargalo/Encargalo.app-api.go/commit/ef9bf740a8144dd2eef6a7b468a00ec1a3ba1f30))
* **shops:** agregar información de distancia y tiempo a la respuesta de tiendas, incluyendo cálculo de tarifas de entrega ([c32b43b](https://github.com/Encargalo/Encargalo.app-api.go/commit/c32b43be2cd8327dff9ade3e94b268ec4df39ae3))
* **shops:** mejorar manejo de errores al obtener negocios y agregar validación de resultados vacíos ([bb6945c](https://github.com/Encargalo/Encargalo.app-api.go/commit/bb6945c7f27c08f708b9792633a3bd1e35013b24))
* **shops:** mejorar manejo de errores al obtener negocios y agregar validación de resultados vacíos ([e8a44f3](https://github.com/Encargalo/Encargalo.app-api.go/commit/e8a44f395ad540e458189fe1d43c2c6108bf0991))
* **shops:** se agg la logica para activar y desactivar el limite de busqueda de los negocios. ([b10d67b](https://github.com/Encargalo/Encargalo.app-api.go/commit/b10d67b8b974baad68bc8fa545cd265f3aa3eadf))
* **shops:** se agg la logica para activar y desactivar el limite de busqueda de los negocios. ([4a30948](https://github.com/Encargalo/Encargalo.app-api.go/commit/4a3094887a1e416718e5ccb17bab6330982d6e4f))
* **shops:** se agregó un cache de 2 minutos para los negocios ([549ca2f](https://github.com/Encargalo/Encargalo.app-api.go/commit/549ca2fb741ca4b26acdd94bd3f752a551a19365))
* **shops:** se agregó un cache de 2 minutos para los negocios ([aef6601](https://github.com/Encargalo/Encargalo.app-api.go/commit/aef660173f0a1da7b641afbca2f8fd2e13033488))


### Bug Fixes

* actualizar la versión del paquete gosuite a v1.0.3 ([1d23484](https://github.com/Encargalo/Encargalo.app-api.go/commit/1d234846a6d701c470ec82e48ecabd7280bb5c63))
* actualizar la versión del paquete gosuite a v1.0.3 ([eaede34](https://github.com/Encargalo/Encargalo.app-api.go/commit/eaede34b3218c352339e78f85687b499cf68b09f))
* actualizar manejo de conexiones WebSocket para usar WSClient y mejorar la gestión de registros ([72acc66](https://github.com/Encargalo/Encargalo.app-api.go/commit/72acc6641dcf7f4270c7ff64de7ff75c30edffc0))
* **auth:** corregir el nombre del campo 'phone' a 'phone_number' en SignInRequest ([8ec6243](https://github.com/Encargalo/Encargalo.app-api.go/commit/8ec62436f8ab5a11c3bc22595fb67e9a6b7a8eac))
* **ci:** actualizar versión de Go a 1.25 y mejorar verificación de pruebas ([c029602](https://github.com/Encargalo/Encargalo.app-api.go/commit/c029602c0049b870071af627993a9b23c420785f))
* **config:** se agg puntero a la variable de entorno de limit_status ([12d9b75](https://github.com/Encargalo/Encargalo.app-api.go/commit/12d9b75f7baaef539f8634d51edc912af5bc22e8))
* **config:** se agregó la etiqueta validate, a la consifuración para que no levante si falta alguna variable de entorno. ([5bcf67f](https://github.com/Encargalo/Encargalo.app-api.go/commit/5bcf67ff0fe324dca69d739ce95e710888ca65ff))
* **config:** se agregó la etiqueta validate, a la consifuración para que no levante si falta alguna variable de entorno. ([05deb57](https://github.com/Encargalo/Encargalo.app-api.go/commit/05deb5737955da7b979c020d6f5b813f9be18e35))
* corregir campos de la estructura Address para alinearse con MongoDB ([c432d74](https://github.com/Encargalo/Encargalo.app-api.go/commit/c432d74a32ee4540a39d5a3b3a23ed47bedcd8f3))
* corregir rutas de grupos de órdenes para incluir la barra inicial ([a7012e8](https://github.com/Encargalo/Encargalo.app-api.go/commit/a7012e85ab5aa782c47b5f5743cf70cd7ab3856f))
* **customers:** manejar error de búsqueda de cliente con constante errcustom ([674f880](https://github.com/Encargalo/Encargalo.app-api.go/commit/674f88037eda3831415319bfbf035511953a0d3a))
* mejorar manejo de conexiones WebSocket y optimizar registro de órdenes ([c73e8bf](https://github.com/Encargalo/Encargalo.app-api.go/commit/c73e8bf494e0410044e83d654a44c5a8b9bf272b))
* mejorar manejo de errores y registro de órdenes en la función Register ([61a8246](https://github.com/Encargalo/Encargalo.app-api.go/commit/61a824624a6dc8186e0c6a4434199aba43a7253e))
* **order:** ajustar cálculo del precio total para incluir la tarifa de entrega ([f7cf228](https://github.com/Encargalo/Encargalo.app-api.go/commit/f7cf2283cb76b013ea38a62d42b640bee9d8ffad))
* **release:** correct condition for docker job execution ([e3e161f](https://github.com/Encargalo/Encargalo.app-api.go/commit/e3e161fe53041b627a9c989f419e5ad5aefff6d8))
* **release:** correct condition for docker job execution ([1521d1f](https://github.com/Encargalo/Encargalo.app-api.go/commit/1521d1f61cd80795ef7ec25db749e1ff10bb4a92))
* **router:** corregir importación de documentación en el enrutador ([82e39b6](https://github.com/Encargalo/Encargalo.app-api.go/commit/82e39b648de9c72416e65403f525486a00b031bf))
* target develop branch for release-please releases ([274c377](https://github.com/Encargalo/Encargalo.app-api.go/commit/274c377588cbe77c15332d22af3bc494f65be409))
* trigger release-please on both develop and main PR merges ([28c88d0](https://github.com/Encargalo/Encargalo.app-api.go/commit/28c88d0712f456224d5201714108f5817ee724f9))
* update import path for gosuite config package ([23eef79](https://github.com/Encargalo/Encargalo.app-api.go/commit/23eef79f6e90f0bc729280b025b64dfb6a8a2bf9))
* update import path for gosuite config package ([b901cf0](https://github.com/Encargalo/Encargalo.app-api.go/commit/b901cf0c5157c0053d83f0e0098bc6035b22a262))
* use PAT to allow release-please to create pull requests ([6ec5256](https://github.com/Encargalo/Encargalo.app-api.go/commit/6ec52568a961adf2a845476de3878a016bd0d23f))
* use PAT to allow release-please to create pull requests ([91d47f9](https://github.com/Encargalo/Encargalo.app-api.go/commit/91d47f9b26c0dd150296f7b270ae269441b1b10b))


### Refactors

* **auth:** eliminar validaciones de rol de usuario en manejo de direcciones y empleados ([0be5727](https://github.com/Encargalo/Encargalo.app-api.go/commit/0be5727b19248eb76ab6220a500ba0f3254d35dd))
* consolidate employee phone search into users domain ([703a43d](https://github.com/Encargalo/Encargalo.app-api.go/commit/703a43d6d2b001d9f99330c64b14fd1653ee2d99))
* eliminar claves de contexto innecesarias en el middleware de autenticación ([65c8626](https://github.com/Encargalo/Encargalo.app-api.go/commit/65c8626bbbee2a4d812810bc96d18b5b3d42596a))
* **event:** unify single and batch event publishing ([bdda62a](https://github.com/Encargalo/Encargalo.app-api.go/commit/bdda62aad6e528748aa8b50682040a8774cb1395))
* improve error handling and update HTTP error responses across multiple handlers ([d2b433b](https://github.com/Encargalo/Encargalo.app-api.go/commit/d2b433b6b6a26d1f2410918d7f9fc08d3a74453c))
* improve error handling by converting errors to HTTP errors in multiple handlers ([b14e507](https://github.com/Encargalo/Encargalo.app-api.go/commit/b14e50700b953cf98d22f3ef164b1f08e9231cc4))
* remove deprecated routes and add Postman collections ([3726262](https://github.com/Encargalo/Encargalo.app-api.go/commit/3726262d058614cbafcb47ff72d8d8a0850a0433))
* rename customers domain to users and unify Account model ([b6b9574](https://github.com/Encargalo/Encargalo.app-api.go/commit/b6b957430fefdc19b3a810969b2a8a7e1df2942d))
* rename customers domain to users and update routes to /users ([93ab611](https://github.com/Encargalo/Encargalo.app-api.go/commit/93ab611d88ffd9e325a3ddd1e80448b90c0b7edf))
* **shops:** eliminar comentarios innecesarios en GetShopsBy y calculateDistances ([9f5d852](https://github.com/Encargalo/Encargalo.app-api.go/commit/9f5d852e516c87aa14ac560ca3d164a5238f26ed))
* **shops:** eliminar comentarios innecesarios en GetShopsBy y calculateDistances ([669af55](https://github.com/Encargalo/Encargalo.app-api.go/commit/669af558ebc0aa4fbffa7507ed853ce8d8e10886))
* update Go version to 1.26 and replace 'interface{}' with 'any' in multiple files ([776a5a6](https://github.com/Encargalo/Encargalo.app-api.go/commit/776a5a62f05d695fc41b7ed8f13978f728962cd8))

## [1.4.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.3.0...v1.4.0) (2026-03-24)


### Features

* **release:** add Docker publish workflow after release creation ([e90899b](https://github.com/Encargalo/Encargalo.app-api.go/commit/e90899b31e991eb436d4085125ace832be96029c))
* **release:** add Docker publish workflow after release creation ([34fe886](https://github.com/Encargalo/Encargalo.app-api.go/commit/34fe886017698bdbbdc993fc79b69eebe1ad27f7))

## [1.3.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.2.0...v1.3.0) (2026-03-24)


### Features

* **products:** add pagination to all product listing endpoints ([d18649c](https://github.com/Encargalo/Encargalo.app-api.go/commit/d18649c7e45569725266b235eee2fbcae8f0c390))
* **products:** add pagination to all product listing endpoints ([754d66a](https://github.com/Encargalo/Encargalo.app-api.go/commit/754d66a4e73b8f15709d510acb55957eff1799bc))

## [1.2.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.1.0...v1.2.0) (2026-03-23)


### Features

* (shops): se agregó logica condicional para que no se calcule distancia ni tiempo al consultar los negpocios. ([0ea91b1](https://github.com/Encargalo/Encargalo.app-api.go/commit/0ea91b18e676651cc5edacd0af0d13360b3d4209))
* add dockerfile ([bc3a427](https://github.com/Encargalo/Encargalo.app-api.go/commit/bc3a427b5d45ca3bc95185eb4cf991928dc74a25))
* Add new product-related endpoints and data structures ([7e9c746](https://github.com/Encargalo/Encargalo.app-api.go/commit/7e9c7468b2d1787a89a506ca71342f163d3c34ae))
* Add new product-related endpoints and data structures ([378f398](https://github.com/Encargalo/Encargalo.app-api.go/commit/378f39854e381b3fd22bbeb7b3f3550d170da589))
* add peticion post ([b61dd07](https://github.com/Encargalo/Encargalo.app-api.go/commit/b61dd07a28c24a52926f5dd94e8fac638675ffdd))
* add release automation and Docker publish workflows ([543b8ae](https://github.com/Encargalo/Encargalo.app-api.go/commit/543b8aed133c9eaae3919cbc1309c357484cb5ae))
* add release automation and Docker publish workflows ([dc322c6](https://github.com/Encargalo/Encargalo.app-api.go/commit/dc322c6c3a8030882cb36af35506fb3172fe1ee4))
* **address:** se cambio la base de datos de las direcciones de mongodb a postgres. ([c8395b1](https://github.com/Encargalo/Encargalo.app-api.go/commit/c8395b1f40730d0062ddc3e9b049fbf02ee7515e))
* **address:** se eliminió la columna de reference de la base de datos. ([2c4dd43](https://github.com/Encargalo/Encargalo.app-api.go/commit/2c4dd43f7b3d231d655fe31e1523b184c051d072))
* agregar campo OrderID en DataEventOrderCreated y ajustar lógica en BuidlToModelEvent ([a4a1a1a](https://github.com/Encargalo/Encargalo.app-api.go/commit/a4a1a1ad6c3a89d009f0a2ba5a8b2acf283df0f7))
* agregar campo OrderID en DataEventOrderCreated y ajustar lógica en BuidlToModelEvent ([ce3d591](https://github.com/Encargalo/Encargalo.app-api.go/commit/ce3d591df318f3f8c968ee95820bfe97725b784a))
* agregar manejo de cierre de sesión para clientes y empleados, actualizando la lógica de cookies ([f2b2882](https://github.com/Encargalo/Encargalo.app-api.go/commit/f2b2882459cacaf1b280b6a07dc1e4e4cc7dbe74))
* agregar manejo de empleados con nuevas rutas y DTOs ([694f5fd](https://github.com/Encargalo/Encargalo.app-api.go/commit/694f5fdc26eb90210da854917fae94a02eaf8d95))
* agregar manejo de empleados con nuevas rutas y DTOs ([879b45c](https://github.com/Encargalo/Encargalo.app-api.go/commit/879b45c6273aef82e908f3319fc102628c45a5e1))
* agregar manejo del entorno en la configuración y ajustar lógica de consulta en producción ([cf57b94](https://github.com/Encargalo/Encargalo.app-api.go/commit/cf57b944598f67352a531d32a92ff8622cb3b22c))
* agregar nombre y teléfono del cliente en el modelo de orden y en la aplicación de órdenes ([35cc36c](https://github.com/Encargalo/Encargalo.app-api.go/commit/35cc36cce576d8d51dfa9a000869d25b121da2f7))
* agregar nombre y teléfono del cliente en el modelo de orden y en la aplicación de órdenes ([c380308](https://github.com/Encargalo/Encargalo.app-api.go/commit/c380308063481f0795ac3b7a7f7571a906463c1b))
* ajustar lógica de consulta para entornos de producción en GetAllShops y GetShopsBy ([8c7f618](https://github.com/Encargalo/Encargalo.app-api.go/commit/8c7f618ac1993034dca6a83d889eb6deb4e56d03))
* **auth, employee, orders:** agregar autenticación de empleados y gestión de órdenes por WebSocket ([44fc07e](https://github.com/Encargalo/Encargalo.app-api.go/commit/44fc07eb34076109fb51e73a18db1aae794977b9))
* **auth:** add JWT session management and refactor authentication handler ([6182b27](https://github.com/Encargalo/Encargalo.app-api.go/commit/6182b27563eab6dc06ea8a9f8c2d70e5c6e8c011))
* **auth:** add session search functionality and update session management logic ([d537955](https://github.com/Encargalo/Encargalo.app-api.go/commit/d537955aa512cc46ad3e8174594bcc422a3b8f1f))
* **auth:** agregar endpoints para inicio de sesión de clientes y empleados ([8e1d626](https://github.com/Encargalo/Encargalo.app-api.go/commit/8e1d6262feb89b6545554f3343a807184de265aa))
* **auth:** agregar funcionalidad para eliminar sesiones activas y actualizar la documentación de la API ([32bf0a2](https://github.com/Encargalo/Encargalo.app-api.go/commit/32bf0a277c1f2589f86c6a5c2b844bd42e53a5fb))
* **auth:** agregar funcionalidad para eliminar sesiones activas y actualizar la documentación de la API ([bb35754](https://github.com/Encargalo/Encargalo.app-api.go/commit/bb35754653835e6c118620a7f71992acd28a21e8))
* **auth:** enhance authentication middleware with session validation ([d537955](https://github.com/Encargalo/Encargalo.app-api.go/commit/d537955aa512cc46ad3e8174594bcc422a3b8f1f))
* **auth:** implement customer sign-in functionality with JWT support ([6155e8d](https://github.com/Encargalo/Encargalo.app-api.go/commit/6155e8d1e7c987fd2f5be795cf2a149d49ac3596))
* **auth:** integrate Redis for session management and update authentication logic ([21f506a](https://github.com/Encargalo/Encargalo.app-api.go/commit/21f506a4b73f6c86f9aa318cf5f32e0abeade31d))
* **auth:** mejorar manejo de sesiones y roles; actualizar middleware y modelos ([db9bef1](https://github.com/Encargalo/Encargalo.app-api.go/commit/db9bef170a6102078de5134d331109b988bb0dc5))
* **auth:** refactor middleware and session handling; add role management ([fac62c8](https://github.com/Encargalo/Encargalo.app-api.go/commit/fac62c8fe15b230b8e18149b4e8bed477c334d71))
* **auth:** refactor y unificar middleware de autenticación para clientes y empleados ([86de5ad](https://github.com/Encargalo/Encargalo.app-api.go/commit/86de5ad05337d103efcd8c133d0ec6e24a88f563))
* **auth:** unificar manejo de cierre de sesión y actualizar rutas de autenticación ([ca78536](https://github.com/Encargalo/Encargalo.app-api.go/commit/ca78536f198f0cab97159d722e8493510598eac9))
* **CI:** agg archivo de continuos integration. ([729fb54](https://github.com/Encargalo/Encargalo.app-api.go/commit/729fb54c080695a31c71359607b79bb25147c5f3))
* **CI:** agg archivo de continuos integration. ([0916128](https://github.com/Encargalo/Encargalo.app-api.go/commit/0916128f7daf5e340323c51a9894ecd001fb05fc))
* **cookie:** actualizar la política SameSite de las cookies a None p… ([ff3a5d2](https://github.com/Encargalo/Encargalo.app-api.go/commit/ff3a5d2fc657cd87b58451e94daaa7cc939cc1e2))
* **cookie:** actualizar la política SameSite de las cookies a None para permitir el uso en contextos de terceros ([3e6df82](https://github.com/Encargalo/Encargalo.app-api.go/commit/3e6df8237f7a64a56d043ccba43991b611cb4265))
* **cookie:** actualizar la política SameSite de las cookies a Strict para mejorar la seguridad ([49be14f](https://github.com/Encargalo/Encargalo.app-api.go/commit/49be14f912686f886c58a7a9ef238b306dd848df))
* **creditDiscount:** feat(creditDiscount):  ([2a87552](https://github.com/Encargalo/Encargalo.app-api.go/commit/2a8755202c9b57439d02c4cc896bcbb99b9fdc0d))
* **customers:** add customer management endpoints and address handling ([f375f38](https://github.com/Encargalo/Encargalo.app-api.go/commit/f375f38510d18f7ec111c1c12f62ea4fee306c0a))
* **customers:** implement customer group routing and middleware integration ([d537955](https://github.com/Encargalo/Encargalo.app-api.go/commit/d537955aa512cc46ad3e8174594bcc422a3b8f1f))
* **customers:** mejorar la búsqueda de clientes y manejar errores de forma más clara ([da56b5e](https://github.com/Encargalo/Encargalo.app-api.go/commit/da56b5ed9ed59ce900d90672ae32826f732db778))
* **docker:** actualizar la imagen base de Alpine a 3.21 ([34c0101](https://github.com/Encargalo/Encargalo.app-api.go/commit/34c010129813dc3e37cdcd3edb0936abde8973c4))
* **docker:** actualizar la imagen base de Alpine a 3.21 ([d98c42d](https://github.com/Encargalo/Encargalo.app-api.go/commit/d98c42d5f4964a191f3303cdefa5f3d875cabf5a))
* **docker:** actualizar la imagen de Go a 1.25.5 y corregir el puerto expuesto a 3000 ([a5cdcd9](https://github.com/Encargalo/Encargalo.app-api.go/commit/a5cdcd9d0cc601e03c2de5f086d1933374a97460))
* eliminar el grupo de autenticación y ajustar la lógica de cierre de sesión para empleados ([b4fdc52](https://github.com/Encargalo/Encargalo.app-api.go/commit/b4fdc52667fafe1ad99b590900044d4eeec27916))
* **employee:** agregar middleware de autenticación para empleados y métodos de búsqueda por ID ([b872a89](https://github.com/Encargalo/Encargalo.app-api.go/commit/b872a89bf7dca4e957bb6731e0a80df352db83fe))
* enhance products domain with flavors, image update, and orders image snapshot ([11c40cd](https://github.com/Encargalo/Encargalo.app-api.go/commit/11c40cd545a7be556270cabc08eb937a5f197b57))
* enhance products domain with flavors, image update, and orders image snapshot ([f645413](https://github.com/Encargalo/Encargalo.app-api.go/commit/f64541388e55cef29d4a18d15c10b5a01c77aa2b))
* **go:** actualizar la versión de Go a 1.25 y actualizar dependencias ([6ccfc8f](https://github.com/Encargalo/Encargalo.app-api.go/commit/6ccfc8fc85abf8e8271d803869fec0ab82a3c5e5))
* implement shops API with handlers, repository, and Swagger documentation ([c01a8f6](https://github.com/Encargalo/Encargalo.app-api.go/commit/c01a8f62437e76ed9dfd04b56f6e94ad21061597))
* implementar lógica dinámica para el atributo SameSite en las cookies. ([30052e2](https://github.com/Encargalo/Encargalo.app-api.go/commit/30052e28dd8f629f757d6700c98dd75a8060f96c))
* initialize Go project with basic health check and Swagger documentation ([7593bf9](https://github.com/Encargalo/Encargalo.app-api.go/commit/7593bf9ee264b96896976a01aef3ffe9c9c70a78))
* **logging:** integrate Slack logging for error handling and notifications ([6aa2c24](https://github.com/Encargalo/Encargalo.app-api.go/commit/6aa2c24637c27a35f6ff3f84928ff15208930bca))
* **migrations:** agregar columnas customer_name y phone_customer a la tabla de órdenes ([89b2fe0](https://github.com/Encargalo/Encargalo.app-api.go/commit/89b2fe0d37aa8d4d605aa1362c59103bc44db2ad))
* **order:** agregar campo 'number' a la entidad Order y actualizar la lógica de creación de pedidos ([c360166](https://github.com/Encargalo/Encargalo.app-api.go/commit/c360166fe4475f371323fe30eeabcbe8f44b527c))
* **order:** agregar campo 'number' a la entidad Order y actualizar la lógica de creación de pedidos ([ebabb78](https://github.com/Encargalo/Encargalo.app-api.go/commit/ebabb78963dd04fdfa3fbd582c014904a919b402))
* **order:** agregar campo de tarifa de entrega en la creación de pedidos ([58b58b0](https://github.com/Encargalo/Encargalo.app-api.go/commit/58b58b03cfd65f56303a93cc52ec9382549649a7))
* **order:** eliminar el campo 'name' de CreateAdditionsOrders y actualizar la lógica de asignación de nombres en la creación de pedidos ([13cda25](https://github.com/Encargalo/Encargalo.app-api.go/commit/13cda2572ad239aed7e362da2f19f79e43db7d06))
* **order:** eliminar el campo 'name' de CreateItemsOrder y actualizar la lógica de asignación de nombres en la creación de pedidos ([41fd2a6](https://github.com/Encargalo/Encargalo.app-api.go/commit/41fd2a605933306e702beae705a3e423038b3009))
* **order:** incluir relaciones de ItemsOrders y Additions en la búsqueda de pedidos por ID ([b127fd1](https://github.com/Encargalo/Encargalo.app-api.go/commit/b127fd1e5dcea9ef6ab30d61122f1c0f005263aa))
* **order:** refactor UpdateOrderStatus to use typed constants for order statuses and payment methods ([118d7aa](https://github.com/Encargalo/Encargalo.app-api.go/commit/118d7aa03f53ddb14e0ef3a89ff059bbacf59981))
* **order:** refactor UpdateOrderStatus to use typed constants for order statuses and payment methods ([2c35dcc](https://github.com/Encargalo/Encargalo.app-api.go/commit/2c35dcc37c5615932b3a283825608f48ee96c16e))
* **orders, shops:** agregar búsqueda de batch por ID de tienda y act… ([f13207a](https://github.com/Encargalo/Encargalo.app-api.go/commit/f13207a679c04f8df0629218ce3f3edc8743e1d7))
* **orders, shops:** agregar búsqueda de batch por ID de tienda y actualizar modelo de orden ([73da058](https://github.com/Encargalo/Encargalo.app-api.go/commit/73da058f328ca7e8839f93b06b4daf7b943b25ec))
* **orders:** add order management functionality with creation endpoint and data models ([a41031a](https://github.com/Encargalo/Encargalo.app-api.go/commit/a41031af8ba441b87bd3f1008091ccb1739bc536))
* **orders:** add order management functionality with creation endpoint and data models ([5a829c9](https://github.com/Encargalo/Encargalo.app-api.go/commit/5a829c9dcd3754a4ea3a65e9fd474e1d346a95c8))
* **orders:** agregar actualización de estado de órdenes y soporte para múltiples conexiones WebSocket ([5902727](https://github.com/Encargalo/Encargalo.app-api.go/commit/5902727a7d8df44fafdad5d0d3e96188a9d25ba6))
* **orders:** agregar búsqueda de órdenes abiertas por batch y mejorar gestión de conexiones WebSocket ([cbcaf62](https://github.com/Encargalo/Encargalo.app-api.go/commit/cbcaf62a4ff79b29e0f879c6ce08024d94b10465))
* **orders:** agregar evento para la creación de órdenes en el flujo de Redis ([cf8cc8e](https://github.com/Encargalo/Encargalo.app-api.go/commit/cf8cc8eb9af0a152338361059170b5885f88c573))
* **orders:** agregar evento para la creación de órdenes en el flujo de Redis ([b71e531](https://github.com/Encargalo/Encargalo.app-api.go/commit/b71e5311ed5651b21791dc620e71537993fedb30))
* **orders:** agregar manejo de fecha de cancelación al actualizar el estado de la orden ([ae74dcf](https://github.com/Encargalo/Encargalo.app-api.go/commit/ae74dcfe3f885ebb3071c3831a1c0490d2dd4419))
* **orders:** agregar relaciones de ItemsOrders y Additions en la búsqueda de órdenes por batch abierto ([879c76a](https://github.com/Encargalo/Encargalo.app-api.go/commit/879c76a6e9f7625edc8eb566024587a5c01d99b4))
* **orders:** agregar soporte para WebSocket en la gestión de órdenes y actualizar la estructura de datos ([36be452](https://github.com/Encargalo/Encargalo.app-api.go/commit/36be4523283b6fb73abe04dfd28884e94a1ed501))
* **orders:** implement Redis stream integration for order processing ([4d4f447](https://github.com/Encargalo/Encargalo.app-api.go/commit/4d4f44732222dc8f2e2b84aa9f405afcd86229d0))
* **orders:** integrate Redis stream for order processing and add producer functionality ([94e7b5d](https://github.com/Encargalo/Encargalo.app-api.go/commit/94e7b5d9b7ecc1cfa5ccdabe594f1be9bf114d1c))
* **orders:** integrate Redis stream for order processing and add producer functionality ([b8c5843](https://github.com/Encargalo/Encargalo.app-api.go/commit/b8c58434d0fddc56310a12a42bfbfe984c1e5602))
* **orders:** mejorar gestión de errores en la búsqueda de órdenes por batch abierto y optimizar el manejo de conexiones WebSocket ([b63f7c3](https://github.com/Encargalo/Encargalo.app-api.go/commit/b63f7c30d14a59c6f2edf84fa4cb8e0730471084))
* **orders:** se agg lógica para implementar el campo details en la base de datos. ([0a9e730](https://github.com/Encargalo/Encargalo.app-api.go/commit/0a9e730075210975df79338c5fb309e87f72fc7c))
* **orders:** se agg peticion http para envio de peticion para descontar el % de los creditos de los negocios. ([d4ce708](https://github.com/Encargalo/Encargalo.app-api.go/commit/d4ce708351727968e8b0ff872d7ec5443c8f729d))
* **orders:** se liminaron los campos de customer_nae y phone_customer de las ordenes, también se agrego la migración correspondiente. ([24bf03e](https://github.com/Encargalo/Encargalo.app-api.go/commit/24bf03e62bfb3731a49b2758704a6fd5ec27bf02))
* **postgres:** add migrations system ([dc009bb](https://github.com/Encargalo/Encargalo.app-api.go/commit/dc009bba5aa8ebaf57d6d1e02131319159d87c04))
* **products:** actualizar manejo de imágenes en el registro de productos; incluir ID y hash de imagen ([5cc8766](https://github.com/Encargalo/Encargalo.app-api.go/commit/5cc87664393951efc4dc11066874046b7f81b07c))
* **products:** corregir manejo de ID público de imagen y actualizar código de estado de respuesta al registrar productos ([2ff2977](https://github.com/Encargalo/Encargalo.app-api.go/commit/2ff29770ee1bb0f6fc112b199e759acd21282b3d))
* **products:** implementar actualización de productos con validación y manejo de errores ([e77b348](https://github.com/Encargalo/Encargalo.app-api.go/commit/e77b348cad36e10f2370c4915c9ed1eda5be0205))
* **products:** implementar registro de productos con carga de imagen y validación ([ab13928](https://github.com/Encargalo/Encargalo.app-api.go/commit/ab1392887b5c2d9e5fe79eec7368ddbb511b950c))
* **products:** mejorar manejo de errores al registrar productos; agregar errores específicos para carga y conversión de imágenes ([bb9fefa](https://github.com/Encargalo/Encargalo.app-api.go/commit/bb9fefa5c3e706cef97618d73f6d824baa3f9b73))
* **products:** se agregó un cashe de una hora para los productos de cada negocio ([09238ce](https://github.com/Encargalo/Encargalo.app-api.go/commit/09238cebfa2f05e4feff1148b8f2a2e16aa0a9c8))
* **products:** se agregó un cashe de una hora para los productos de cada negocio ([ea0f71d](https://github.com/Encargalo/Encargalo.app-api.go/commit/ea0f71d168733b5a6419e754cb3963347fad4e6d))
* **products:** se mejoraron las respuesta de los errores en el handler ([c58c64b](https://github.com/Encargalo/Encargalo.app-api.go/commit/c58c64bdc51a43350d55bd040f914e75bbdd69f0))
* refactor manejo de cookies y sesiones, eliminando dependencias innecesarias y mejorando la lógica de creación y eliminación de cookies ([e50b7d8](https://github.com/Encargalo/Encargalo.app-api.go/commit/e50b7d8df5b07cedd22494b1995c8c2c97e87239))
* reorganizar grupos de rutas para pedidos de clientes y empleados ([6e9202c](https://github.com/Encargalo/Encargalo.app-api.go/commit/6e9202c19d9b062db9feebf0f8095d07189f7d9d))
* reorganizar grupos de rutas para pedidos de clientes y empleados ([5b9fe01](https://github.com/Encargalo/Encargalo.app-api.go/commit/5b9fe0155c17622d891f9bda30bce7c83f944051))
* se agg logica para inciar sesión cuando se registra un nuevo customer ([9956f9d](https://github.com/Encargalo/Encargalo.app-api.go/commit/9956f9dd7f680ec27e28e795f764e97ee6c78e0c))
* se agg logica para inciar sesión cuando se registra un nuevo customer ([50407d0](https://github.com/Encargalo/Encargalo.app-api.go/commit/50407d01b3800ce0f45e39a574c70e3253b7209e))
* **shops:** agregar información de distancia y tiempo a la respuesta de tiendas, incluyendo cálculo de tarifas de entrega ([ef9bf74](https://github.com/Encargalo/Encargalo.app-api.go/commit/ef9bf740a8144dd2eef6a7b468a00ec1a3ba1f30))
* **shops:** agregar información de distancia y tiempo a la respuesta de tiendas, incluyendo cálculo de tarifas de entrega ([c32b43b](https://github.com/Encargalo/Encargalo.app-api.go/commit/c32b43be2cd8327dff9ade3e94b268ec4df39ae3))
* **shops:** mejorar manejo de errores al obtener negocios y agregar validación de resultados vacíos ([bb6945c](https://github.com/Encargalo/Encargalo.app-api.go/commit/bb6945c7f27c08f708b9792633a3bd1e35013b24))
* **shops:** mejorar manejo de errores al obtener negocios y agregar validación de resultados vacíos ([e8a44f3](https://github.com/Encargalo/Encargalo.app-api.go/commit/e8a44f395ad540e458189fe1d43c2c6108bf0991))
* **shops:** se agg la logica para activar y desactivar el limite de busqueda de los negocios. ([b10d67b](https://github.com/Encargalo/Encargalo.app-api.go/commit/b10d67b8b974baad68bc8fa545cd265f3aa3eadf))
* **shops:** se agg la logica para activar y desactivar el limite de busqueda de los negocios. ([4a30948](https://github.com/Encargalo/Encargalo.app-api.go/commit/4a3094887a1e416718e5ccb17bab6330982d6e4f))
* **shops:** se agregó un cache de 2 minutos para los negocios ([549ca2f](https://github.com/Encargalo/Encargalo.app-api.go/commit/549ca2fb741ca4b26acdd94bd3f752a551a19365))
* **shops:** se agregó un cache de 2 minutos para los negocios ([aef6601](https://github.com/Encargalo/Encargalo.app-api.go/commit/aef660173f0a1da7b641afbca2f8fd2e13033488))


### Bug Fixes

* actualizar la versión del paquete gosuite a v1.0.3 ([1d23484](https://github.com/Encargalo/Encargalo.app-api.go/commit/1d234846a6d701c470ec82e48ecabd7280bb5c63))
* actualizar la versión del paquete gosuite a v1.0.3 ([eaede34](https://github.com/Encargalo/Encargalo.app-api.go/commit/eaede34b3218c352339e78f85687b499cf68b09f))
* actualizar manejo de conexiones WebSocket para usar WSClient y mejorar la gestión de registros ([72acc66](https://github.com/Encargalo/Encargalo.app-api.go/commit/72acc6641dcf7f4270c7ff64de7ff75c30edffc0))
* **auth:** corregir el nombre del campo 'phone' a 'phone_number' en SignInRequest ([8ec6243](https://github.com/Encargalo/Encargalo.app-api.go/commit/8ec62436f8ab5a11c3bc22595fb67e9a6b7a8eac))
* **ci:** actualizar versión de Go a 1.25 y mejorar verificación de pruebas ([c029602](https://github.com/Encargalo/Encargalo.app-api.go/commit/c029602c0049b870071af627993a9b23c420785f))
* **config:** se agg puntero a la variable de entorno de limit_status ([12d9b75](https://github.com/Encargalo/Encargalo.app-api.go/commit/12d9b75f7baaef539f8634d51edc912af5bc22e8))
* **config:** se agregó la etiqueta validate, a la consifuración para que no levante si falta alguna variable de entorno. ([5bcf67f](https://github.com/Encargalo/Encargalo.app-api.go/commit/5bcf67ff0fe324dca69d739ce95e710888ca65ff))
* **config:** se agregó la etiqueta validate, a la consifuración para que no levante si falta alguna variable de entorno. ([05deb57](https://github.com/Encargalo/Encargalo.app-api.go/commit/05deb5737955da7b979c020d6f5b813f9be18e35))
* corregir campos de la estructura Address para alinearse con MongoDB ([c432d74](https://github.com/Encargalo/Encargalo.app-api.go/commit/c432d74a32ee4540a39d5a3b3a23ed47bedcd8f3))
* corregir rutas de grupos de órdenes para incluir la barra inicial ([a7012e8](https://github.com/Encargalo/Encargalo.app-api.go/commit/a7012e85ab5aa782c47b5f5743cf70cd7ab3856f))
* **customers:** manejar error de búsqueda de cliente con constante errcustom ([674f880](https://github.com/Encargalo/Encargalo.app-api.go/commit/674f88037eda3831415319bfbf035511953a0d3a))
* mejorar manejo de conexiones WebSocket y optimizar registro de órdenes ([c73e8bf](https://github.com/Encargalo/Encargalo.app-api.go/commit/c73e8bf494e0410044e83d654a44c5a8b9bf272b))
* mejorar manejo de errores y registro de órdenes en la función Register ([61a8246](https://github.com/Encargalo/Encargalo.app-api.go/commit/61a824624a6dc8186e0c6a4434199aba43a7253e))
* **order:** ajustar cálculo del precio total para incluir la tarifa de entrega ([f7cf228](https://github.com/Encargalo/Encargalo.app-api.go/commit/f7cf2283cb76b013ea38a62d42b640bee9d8ffad))
* **router:** corregir importación de documentación en el enrutador ([82e39b6](https://github.com/Encargalo/Encargalo.app-api.go/commit/82e39b648de9c72416e65403f525486a00b031bf))
* target develop branch for release-please releases ([274c377](https://github.com/Encargalo/Encargalo.app-api.go/commit/274c377588cbe77c15332d22af3bc494f65be409))
* trigger release-please on both develop and main PR merges ([28c88d0](https://github.com/Encargalo/Encargalo.app-api.go/commit/28c88d0712f456224d5201714108f5817ee724f9))
* update import path for gosuite config package ([23eef79](https://github.com/Encargalo/Encargalo.app-api.go/commit/23eef79f6e90f0bc729280b025b64dfb6a8a2bf9))
* update import path for gosuite config package ([b901cf0](https://github.com/Encargalo/Encargalo.app-api.go/commit/b901cf0c5157c0053d83f0e0098bc6035b22a262))
* use PAT to allow release-please to create pull requests ([6ec5256](https://github.com/Encargalo/Encargalo.app-api.go/commit/6ec52568a961adf2a845476de3878a016bd0d23f))
* use PAT to allow release-please to create pull requests ([91d47f9](https://github.com/Encargalo/Encargalo.app-api.go/commit/91d47f9b26c0dd150296f7b270ae269441b1b10b))


### Refactors

* **auth:** eliminar validaciones de rol de usuario en manejo de direcciones y empleados ([0be5727](https://github.com/Encargalo/Encargalo.app-api.go/commit/0be5727b19248eb76ab6220a500ba0f3254d35dd))
* consolidate employee phone search into users domain ([703a43d](https://github.com/Encargalo/Encargalo.app-api.go/commit/703a43d6d2b001d9f99330c64b14fd1653ee2d99))
* eliminar claves de contexto innecesarias en el middleware de autenticación ([65c8626](https://github.com/Encargalo/Encargalo.app-api.go/commit/65c8626bbbee2a4d812810bc96d18b5b3d42596a))
* **event:** unify single and batch event publishing ([bdda62a](https://github.com/Encargalo/Encargalo.app-api.go/commit/bdda62aad6e528748aa8b50682040a8774cb1395))
* improve error handling and update HTTP error responses across multiple handlers ([d2b433b](https://github.com/Encargalo/Encargalo.app-api.go/commit/d2b433b6b6a26d1f2410918d7f9fc08d3a74453c))
* improve error handling by converting errors to HTTP errors in multiple handlers ([b14e507](https://github.com/Encargalo/Encargalo.app-api.go/commit/b14e50700b953cf98d22f3ef164b1f08e9231cc4))
* remove deprecated routes and add Postman collections ([3726262](https://github.com/Encargalo/Encargalo.app-api.go/commit/3726262d058614cbafcb47ff72d8d8a0850a0433))
* rename customers domain to users and unify Account model ([b6b9574](https://github.com/Encargalo/Encargalo.app-api.go/commit/b6b957430fefdc19b3a810969b2a8a7e1df2942d))
* rename customers domain to users and update routes to /users ([93ab611](https://github.com/Encargalo/Encargalo.app-api.go/commit/93ab611d88ffd9e325a3ddd1e80448b90c0b7edf))
* **shops:** eliminar comentarios innecesarios en GetShopsBy y calculateDistances ([9f5d852](https://github.com/Encargalo/Encargalo.app-api.go/commit/9f5d852e516c87aa14ac560ca3d164a5238f26ed))
* **shops:** eliminar comentarios innecesarios en GetShopsBy y calculateDistances ([669af55](https://github.com/Encargalo/Encargalo.app-api.go/commit/669af558ebc0aa4fbffa7507ed853ce8d8e10886))
* update Go version to 1.26 and replace 'interface{}' with 'any' in multiple files ([776a5a6](https://github.com/Encargalo/Encargalo.app-api.go/commit/776a5a62f05d695fc41b7ed8f13978f728962cd8))
