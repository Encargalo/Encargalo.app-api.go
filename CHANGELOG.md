# Changelog

## [2.2.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v2.1.0...v2.2.0) (2026-10-05)


### Features

* **maps:** devolver todos los resultados del geocoding inverso ([38ad2c8](https://github.com/Encargalo/Encargalo.app-api.go/commit/38ad2c8c256bfcf5517212ff0f1061b1759da29c))
* **products:** incluir macro categoría en el catálogo de productos ([19b8add](https://github.com/Encargalo/Encargalo.app-api.go/commit/19b8adde468d9087cd06a20a4af340f3d09f32fd))
* **products:** permitir precios con centavos en productos y pedidos ([0cefada](https://github.com/Encargalo/Encargalo.app-api.go/commit/0cefada12cad2679769bb69f10f5919006e2a0a8))


### Bug Fixes

* **products:** excluir negocios Turbo del trending ([8c7efe8](https://github.com/Encargalo/Encargalo.app-api.go/commit/8c7efe89a98bd5473e2be7f76f7ef674c220f3b0))
* **products:** excluir negocios Turbo del trending ([8490f2f](https://github.com/Encargalo/Encargalo.app-api.go/commit/8490f2f844919be6555ddb270a117fee3acaf6f7))
* **products:** radio de cobertura y sin Turbo en trending ([2fce84c](https://github.com/Encargalo/Encargalo.app-api.go/commit/2fce84c29efb7bfe0ab87606b96d420ce4a43a42))
* **products:** usar el radio de cobertura de negocios en trending ([dd043ab](https://github.com/Encargalo/Encargalo.app-api.go/commit/dd043ab469fd711363edeaf51024a7afc4e0d645))
* **products:** usar el radio de cobertura de negocios en trending ([6b1e672](https://github.com/Encargalo/Encargalo.app-api.go/commit/6b1e6723c987cd778845809b2d6be5c3efa8a6af))
* **trips:** reembolsar completo los viajes con pago mixto ([b26df39](https://github.com/Encargalo/Encargalo.app-api.go/commit/b26df391143823aceff3fac3fed8aa107f31b561))
* **trips:** reembolsar completo los viajes con pago mixto ([9bafa1b](https://github.com/Encargalo/Encargalo.app-api.go/commit/9bafa1b058fa9ce1903656f532f911e93be4ad9f))
* **trips:** reembolsar completo los viajes con pago mixto ([587ba99](https://github.com/Encargalo/Encargalo.app-api.go/commit/587ba9979f0ed199f390351a5ba3394d3e721254))

## [2.1.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v2.0.1...v2.1.0) (2026-10-01)


### Features

* **riders:** enviar la distancia tienda→entrega en los eventos de orden ([76d8fbd](https://github.com/Encargalo/Encargalo.app-api.go/commit/76d8fbdb043b6f22d48ef6e933ed0f6c902a58f5))
* **riders:** enviar la distancia tienda→entrega en los eventos de orden ([2baffe2](https://github.com/Encargalo/Encargalo.app-api.go/commit/2baffe228a5f114a3e3f389f2a64b282c45e7362)), closes [#235](https://github.com/Encargalo/Encargalo.app-api.go/issues/235)
* **riders:** exponer el perfil y el vehículo del conductor en GET /riders/profile ([e759b81](https://github.com/Encargalo/Encargalo.app-api.go/commit/e759b81ce6364a1be2c60f87d7ec5a228fb69291))
* **riders:** exponer el perfil y el vehículo del conductor en GET /riders/profile ([79a0ef9](https://github.com/Encargalo/Encargalo.app-api.go/commit/79a0ef920576d381427eb1113969fed8f17b4428)), closes [#236](https://github.com/Encargalo/Encargalo.app-api.go/issues/236)
* **trips:** avisar al conductor cuando el cliente cancela el viaje ([c46c114](https://github.com/Encargalo/Encargalo.app-api.go/commit/c46c1141137f37471a88e8b7fa67ff1bcd71d684))


### Bug Fixes

* **shops:** evitar pánico al listar negocios sin coordenadas ([86aaa99](https://github.com/Encargalo/Encargalo.app-api.go/commit/86aaa99eb58ac121b286eaa10e676bdfe4ad8771))

## [2.0.1](https://github.com/Encargalo/Encargalo.app-api.go/compare/v2.0.0...v2.0.1) (2026-09-27)


### Bug Fixes

* **ci:** leer el repo con el token del job al sincronizar el tablero ([7eaeda0](https://github.com/Encargalo/Encargalo.app-api.go/commit/7eaeda0bbc281b6630693c29e89bf4ef2a0c5606))

## [2.0.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.9.0...v2.0.0) (2026-09-27)


### ⚠ BREAKING CHANGES

* **finance:** details.payment y el WS de payment review emiten pago_movil_amount_bs en lugar de pago_movil_amount_usd. Complete de devoluciones y el CTA deben usar campos *_bs.
* **finance:** balance y movimientos solo en balance_bs/balance_usd
* **maps:** la variable de entorno GOOGLE_MATRIX pasa a llamarse GOOGLE_SERVER_KEY (mismo valor). Hay que renombrarla en Railway antes de desplegar o el servicio no arranca (config.Google.ServerKey es requerido).
* **auth:** el backend deja de aceptar la cookie encargalo-session directamente en cualquier ruta protegida por Auth/Require. Depende por completo de que Traefik esté sirviendo el tráfico e inyectando X-Encargalo-Session + X-Internal-Gateway-Secret. Solo seguro de desplegar una vez que Traefik ya sirve el 100% del tráfico público (ver railway-traefik) — que es el caso hoy en staging, validado en la sesión de pruebas previa (login real + rutas protegidas vía Traefik, rate limit, IPs reales).
* **finance:** los endpoints de refund/payment-reviews devuelven y esperan amount_bs / refundable_pm_bs / credited_bs / balance_amount_bs en vez de sus equivalentes *_usd. Los clientes que parseen estos campos deben actualizarse.

### Features

* agregar búsqueda de tiendas por categoría macro y proximidad ([dc6ea2a](https://github.com/Encargalo/Encargalo.app-api.go/commit/dc6ea2a6b00e7b32c5d2aca5670862bc2691eb1f))
* agregar búsqueda de tiendas por categoría macro y proximidad ([b667302](https://github.com/Encargalo/Encargalo.app-api.go/commit/b66730290ffd4544c2d8e034dd6b4f56531adb8c))
* agregar endpoint para obtener productos más vendidos cerca del usuario ([3236b8b](https://github.com/Encargalo/Encargalo.app-api.go/commit/3236b8bdbdf8ff331c0623e66ccd5a4645e354c6))
* agregar endpoint para obtener productos más vendidos cerca del usuario ([b1beabf](https://github.com/Encargalo/Encargalo.app-api.go/commit/b1beabfa35f28c219cc66577c851c4d2bddd041f))
* **analytics:** añadir worker ETL OLTP→OLAP para métricas en Grafana ([962d28a](https://github.com/Encargalo/Encargalo.app-api.go/commit/962d28abb2fba4d6a5e98c0aba0041dcf5477bb7))
* **analytics:** exportar logs del ETL a Loki vía OTLP ([44d9868](https://github.com/Encargalo/Encargalo.app-api.go/commit/44d9868669d2abf80a66ac0f9e939d4093fa2722))
* **analytics:** heatmap de zonas para viajes (cube_trip) ([7d23c31](https://github.com/Encargalo/Encargalo.app-api.go/commit/7d23c315ebb82a56de88021449c1ef1fa7839e39))
* **analytics:** heatmap de zonas para viajes (cube_trip) ([27d46d0](https://github.com/Encargalo/Encargalo.app-api.go/commit/27d46d08005c4d0e14b2f6dd8c5c537c058b2611))
* **analytics:** propagar h3_cell de shops y orders al pipeline ETL/OLAP ([8105246](https://github.com/Encargalo/Encargalo.app-api.go/commit/810524627c0d61f19d67f34a01154b752b527748))
* **analytics:** propagar h3_cell de shops y orders al pipeline ETL/OLAP ([19084d1](https://github.com/Encargalo/Encargalo.app-api.go/commit/19084d17c37fbc1f355730b86a8769a65c52f395))
* **auth:** agregar endpoint interno de verificación de sesión para gateway ([d752fc4](https://github.com/Encargalo/Encargalo.app-api.go/commit/d752fc4c19b9718af524cb328983e323f2a06c17))
* **auth:** Deploy B — Auth() confía en el header de Traefik, no en la cookie ([7af5250](https://github.com/Encargalo/Encargalo.app-api.go/commit/7af525074ec952a89f6727a1a4024a92f45e522b))
* **bcv:** exponer ruta GET /bcv/rate protegida por API key ([bc38815](https://github.com/Encargalo/Encargalo.app-api.go/commit/bc38815470d975a532ee24157809fa22cfc1bf96))
* **bcv:** exponer ruta GET /bcv/rate protegida por API key ([1c1ebeb](https://github.com/Encargalo/Encargalo.app-api.go/commit/1c1ebeb970bd60f64c35a829fda91bdb30d1a2ff))
* **bcv:** exponer ruta GET /bcv/rate protegida por API key para clientes externos ([045567d](https://github.com/Encargalo/Encargalo.app-api.go/commit/045567d3a205eb397437c40beb888bd1f2f6f0de))
* **finance:** calcular devoluciones y créditos cash-refundable en Bs ([cc9c227](https://github.com/Encargalo/Encargalo.app-api.go/commit/cc9c22754191e1407634c836c4237f5197ae4adf))
* **finance:** exponer balance y movimientos en Bs y USD ([7e5f423](https://github.com/Encargalo/Encargalo.app-api.go/commit/7e5f4232dc205187ddbc4a2e84241fb62925c4ef))
* **finance:** migrar balances y transacciones de customers/riders a VES con tasa BCV ([fd0bb7d](https://github.com/Encargalo/Encargalo.app-api.go/commit/fd0bb7d9e7c8613ba49bd24948c0738f2c67d17b))
* **finance:** staff payouts for riders and shops with unified module ([fec8d67](https://github.com/Encargalo/Encargalo.app-api.go/commit/fec8d67c8baa2a3913e2f2e2a897ca290ef7932e))
* **geo:** calcular y persistir h3_cell en shops y orders al crear/editar ([4e5ed1c](https://github.com/Encargalo/Encargalo.app-api.go/commit/4e5ed1c8f40dbfb9bbfd8c0d1d871e0197b2df00))
* **geo:** calcular y persistir h3_cell en shops y orders al crear/editar ([f414a75](https://github.com/Encargalo/Encargalo.app-api.go/commit/f414a750ac0094bc03ad4b797462c5c1c48cc88b))
* **maps:** exponer componentes de dirección de Google al cliente ([c8aa5d0](https://github.com/Encargalo/Encargalo.app-api.go/commit/c8aa5d0ace599b1f342538d9a5f37f4726d2dc8a))
* **maps:** proxy backend para Google Places/Geocoding ([d3b73a7](https://github.com/Encargalo/Encargalo.app-api.go/commit/d3b73a7aea8881d8d8524b3e823c03b64ded1d13))
* **observability:** exponer pprof interno y trazar llamadas externas de /orders ([1b868b9](https://github.com/Encargalo/Encargalo.app-api.go/commit/1b868b9c48ca57739f2c14990fb4423c10e93c19))
* **observability:** exportar trazas y logs vía OTLP al stack de Grafana ([c7c2831](https://github.com/Encargalo/Encargalo.app-api.go/commit/c7c2831ac5857d528c3b77b80629108571ccdeb8))
* **observability:** soportar token bearer en el exportador OTLP ([f7a4cf3](https://github.com/Encargalo/Encargalo.app-api.go/commit/f7a4cf37b3d33d95dd2838d2213ed230862d9898))
* **orders,money:** order history endpoint and Bs price rounding ([489162a](https://github.com/Encargalo/Encargalo.app-api.go/commit/489162a187983f6c8457046a46df01be1551e10c))
* **orders,money:** order history endpoint and Bs price rounding ([1adcaca](https://github.com/Encargalo/Encargalo.app-api.go/commit/1adcacad82f3c24867f925b81b6d3dde67ff1d32))
* **orders,riders:** pago movil, codigos retiro/entrega y comisiones ([929f51f](https://github.com/Encargalo/Encargalo.app-api.go/commit/929f51fdbac21e613d9e3b01778d8d7b40e2863c))
* **orders,riders:** pago movil, códigos retiro/entrega y comisiones ([ba5a5b5](https://github.com/Encargalo/Encargalo.app-api.go/commit/ba5a5b5d4c479c9629fd0ee4c4c2d7c23054cfbb))
* **orders,shops,redis:** broadcast ws on delivery, fix stats lost calc and refactor redis stream config ([673708b](https://github.com/Encargalo/Encargalo.app-api.go/commit/673708b72f841960b399226f1d01f645a2df6495))
* **orders:** agregar códigos de confirmación de recogida y entrega ([f6a0c1f](https://github.com/Encargalo/Encargalo.app-api.go/commit/f6a0c1f1b240fcc3993120a6f5ed31428d11f349))
* **orders:** agregar códigos de confirmación de recogida y entrega ([1ba95ad](https://github.com/Encargalo/Encargalo.app-api.go/commit/1ba95adf1f73bffd732d01149bde83ead8ba9eb3))
* **orders:** alertar comprobantes pendientes y auto-rechazo a los 10 min ([e9521bd](https://github.com/Encargalo/Encargalo.app-api.go/commit/e9521bd527aa61ef719ec72c7997630706af7b75))
* **orders:** payment review, quotes, timeout de aceptación y access log ([59456d9](https://github.com/Encargalo/Encargalo.app-api.go/commit/59456d9c06a632295e4c9665633a17f517297b16))
* **payments:** aceptar pago en efectivo en pedidos y viajes ([8890b93](https://github.com/Encargalo/Encargalo.app-api.go/commit/8890b93442028ad01e2c2ba38862b8ac74586fe6))
* **payments:** aceptar pago en efectivo en pedidos y viajes ([5940ec8](https://github.com/Encargalo/Encargalo.app-api.go/commit/5940ec88d730f2730c9d9507560d3a4a648bcbb5))
* **payments:** combinar dos métodos de pago en pedidos y viajes ([8043681](https://github.com/Encargalo/Encargalo.app-api.go/commit/8043681ac28942c92544de21f06666c002cafba6))
* **payments:** combinar dos métodos de pago en pedidos y viajes ([0b42735](https://github.com/Encargalo/Encargalo.app-api.go/commit/0b4273578c4d3b47743c684721927bd46c6592f0))
* **payments:** unificar validación de pagos, balance y reembolsos ([65d48ad](https://github.com/Encargalo/Encargalo.app-api.go/commit/65d48ad71d8af187f81f64e02256cdf4d5e430f2))
* **payments:** unificar validación de pagos, balance y reembolsos ([69c3773](https://github.com/Encargalo/Encargalo.app-api.go/commit/69c377397be83c20c35f01c8ec1b0329b1411301))
* **pkg/geo:** agregar utilidad de indexación H3 ([2ff7f07](https://github.com/Encargalo/Encargalo.app-api.go/commit/2ff7f0735d73df2980773d1be3d05abe71cb7c12))
* **pkg/geo:** agregar utilidad de indexación H3 ([191f3e3](https://github.com/Encargalo/Encargalo.app-api.go/commit/191f3e3e802e37c15f5ddfd3b40fa61c5b23e93c)), closes [#167](https://github.com/Encargalo/Encargalo.app-api.go/issues/167)
* **postgres:** crear extensión pg_stat_statements ([b6035ee](https://github.com/Encargalo/Encargalo.app-api.go/commit/b6035eec35c991ae6b932f543f63b151d2cc0c70))
* **postgres:** loguear queries lentas o con error a Loki ([5b54c62](https://github.com/Encargalo/Encargalo.app-api.go/commit/5b54c624937e2b93095f613535fc4e604a8eaeca))
* **products,orders,shops:** agregar promociones de lanzamiento y des… ([7e4e731](https://github.com/Encargalo/Encargalo.app-api.go/commit/7e4e731fc7a59a5d2a09d732d8449b8025579616))
* **products,orders,shops:** agregar promociones de lanzamiento y descuentos por producto ([95a68e9](https://github.com/Encargalo/Encargalo.app-api.go/commit/95a68e9b0d39295143c5b7616fd7e6f9ff013268))
* **products:** allow deleting shop discount by item id ([00ffcdf](https://github.com/Encargalo/Encargalo.app-api.go/commit/00ffcdf2c1f60c1362fe9ea4b26a894f436766aa))
* **products:** expose shop discount on items and clarify validation errors ([9fef360](https://github.com/Encargalo/Encargalo.app-api.go/commit/9fef360b21dd087b7aa077277a5d7f9be1e4b9ca))
* **products:** rediseñar sabores, adiciones, disponibilidad y logging ([58bc35a](https://github.com/Encargalo/Encargalo.app-api.go/commit/58bc35a1c01115cbe372ac8c444cc78b6605271c))
* **products:** sembrar categorías default por micro-categoría en vez de macro ([7e881b4](https://github.com/Encargalo/Encargalo.app-api.go/commit/7e881b42beb13161a8ee15d3b062b69bd7d1c662))
* **riders,orders,products:** payload de riders con shop/customer/items y delivery estimado en trending ([1ac6758](https://github.com/Encargalo/Encargalo.app-api.go/commit/1ac6758685ae626578a295196d148ac9f76ca456))
* **riders,orders,products:** payload de riders con shop/customer/items y delivery estimado en trending ([704488d](https://github.com/Encargalo/Encargalo.app-api.go/commit/704488d302ff1c37f3471c13343970d12f1f6307))
* **riders,websocket:** cliente WS compartido, keepalive y reconexión con órdenes disponibles ([ecb885c](https://github.com/Encargalo/Encargalo.app-api.go/commit/ecb885ceddfbb33b53f37680ea7c833fc315d30d))
* **riders:** agregar dominio de riders y asignación de órdenes con WebSocket ([35c7b92](https://github.com/Encargalo/Encargalo.app-api.go/commit/35c7b92caec4207fa37cfa6882c0ca22836e521f))
* **riders:** dominio de riders, WebSocket y asignación de órdenes ([27f9052](https://github.com/Encargalo/Encargalo.app-api.go/commit/27f905258d2f006556fca444643cab8ee477d658))
* **riders:** dominio de riders, WebSocket, balance y comisiones ([e1cc9e9](https://github.com/Encargalo/Encargalo.app-api.go/commit/e1cc9e9a018faebc454606ce4e138e8ced6648ea))
* **riders:** modo activo del rider + ubicación en tiempo real (sub-PR 2/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([2b3db1c](https://github.com/Encargalo/Encargalo.app-api.go/commit/2b3db1c74cc5e7e84fde830a77e47d82fd4e9042))
* **riders:** modo activo del rider + ubicación en tiempo real (sub-PR 2/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([0041c33](https://github.com/Encargalo/Encargalo.app-api.go/commit/0041c33823b1d88453768a122d8ac01203559cdc))
* **riders:** reenviar viajes abiertos y viaje activo al conductor ([11c8696](https://github.com/Encargalo/Encargalo.app-api.go/commit/11c86963e15dca276d4f251d984b6b0d9baa8237))
* **riders:** reenviar viajes abiertos y viaje activo al conductor ([6360672](https://github.com/Encargalo/Encargalo.app-api.go/commit/6360672f68a11ed94fc85e6972a0c0ddcd6f8c57))
* **shared:** agregar envoltorio de respuesta v2 opt-in por header ([a370848](https://github.com/Encargalo/Encargalo.app-api.go/commit/a370848be47cba299854d3ef639e33db7243c252))
* **shared:** recortar el stream de Redis a 48h de retención ([069b690](https://github.com/Encargalo/Encargalo.app-api.go/commit/069b690bab25edc90fc7cd4f1e4dc6cc272f1d06))
* shop finance, WS compartido y reconexión de riders con órdenes disponibles ([a98837c](https://github.com/Encargalo/Encargalo.app-api.go/commit/a98837c16169730eb166d923336ec316951b85aa))
* **shops,orders:** gestionar lotes de jornada por apertura/cierre del local ([ea01b17](https://github.com/Encargalo/Encargalo.app-api.go/commit/ea01b173dbec88b8bd2716d4eab38582426d386c))
* **shops,products:** open/close endpoint, delivery_fee_bs en shops y BCV en app layer ([cdf753e](https://github.com/Encargalo/Encargalo.app-api.go/commit/cdf753e81d426c714dd3f52a418392e446d9289f))
* **shops,products:** open/close endpoint, delivery_fee_bs en shops y… ([140f2c1](https://github.com/Encargalo/Encargalo.app-api.go/commit/140f2c19755d432ce63359f6f0fa4083dcf5808e))
* **shops:** add stats endpoints for business dashboard ([a46fc2c](https://github.com/Encargalo/Encargalo.app-api.go/commit/a46fc2cf72e9b90f3ec39cac335157461bf70669))
* **shops:** agregar modalidad Turbo con asignación por geolocalización ([352557a](https://github.com/Encargalo/Encargalo.app-api.go/commit/352557a0d8cacfb76583e6101bf8955474df67df))
* **shops:** agregar onboarding de shops por staff con gestión de catálogo ([f95fe90](https://github.com/Encargalo/Encargalo.app-api.go/commit/f95fe90b848fd21a51ce60fcd7317bd77901e174))
* **shops:** agregar stats de clientes nuevos vs recurrentes ([90c4483](https://github.com/Encargalo/Encargalo.app-api.go/commit/90c4483b639d172188dd38529b7820419447c538))
* **shops:** clientes nuevos vs recurrentes en stats ([5c69ce8](https://github.com/Encargalo/Encargalo.app-api.go/commit/5c69ce85b200ed5115a167f42704fd0391785ff3))
* **shops:** sistema de balance, comisiones y liquidación para locales ([3c5bfbd](https://github.com/Encargalo/Encargalo.app-api.go/commit/3c5bfbd2a86db0d3827eec7d7a6ab146fa2a4d4c)), closes [#131](https://github.com/Encargalo/Encargalo.app-api.go/issues/131)
* **shops:** soportar tipo Supermarket y filtro opcional por type ([2b9f523](https://github.com/Encargalo/Encargalo.app-api.go/commit/2b9f52318c76431ef1115731eb7995e02e96b15e))
* **tracking:** registrar pixels genéricos y sincronizarlos a OLAP ([22c5b2d](https://github.com/Encargalo/Encargalo.app-api.go/commit/22c5b2d7bc821aa3104c5569a0350a5cac524cf2))
* **trips:** acreditar el pago móvil a la cartera si el viaje no se presta ([3d82feb](https://github.com/Encargalo/Encargalo.app-api.go/commit/3d82feb49ae4a5dc41e06a28b7894b4f3e31b837))
* **trips:** acreditar el pago móvil si el viaje no se presta ([b3b598d](https://github.com/Encargalo/Encargalo.app-api.go/commit/b3b598d8019ad3f3a75b46c44df366e2dd0e0330))
* **trips:** agregar cotización de tarifa con precio fijo desde la creación ([4fa990a](https://github.com/Encargalo/Encargalo.app-api.go/commit/4fa990a166963800c05403a74197a820bd337300))
* **trips:** ciclo de vida completo + RecordTripEarning (sub-PR 4/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([e988c00](https://github.com/Encargalo/Encargalo.app-api.go/commit/e988c00673435839380d14f1d9ad10fb0fc8b839))
* **trips:** ciclo de vida completo + RecordTripEarning (sub-PR 4/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([7f3036d](https://github.com/Encargalo/Encargalo.app-api.go/commit/7f3036d139897d69f9e62ec8a6b82e8742ab5f30))
* **trips:** dominio base de viajes con REST CRUD (sub-PR 1/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([9e6c1c6](https://github.com/Encargalo/Encargalo.app-api.go/commit/9e6c1c6fc4ef455e2e6ffac156ec13fce4111498))
* **trips:** dominio base de viajes con REST CRUD (sub-PR 1/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([c7f3d2a](https://github.com/Encargalo/Encargalo.app-api.go/commit/c7f3d2a622a52930a5382d4451a141ff5ea98d9a))
* **trips:** exigir comprobante de pago móvil antes de ofertar el viaje ([fc14c25](https://github.com/Encargalo/Encargalo.app-api.go/commit/fc14c25912cc182b4ae9f831e6786c3f043dfe0d))
* **trips:** exigir comprobante de pago móvil antes de ofertar el viaje ([e413912](https://github.com/Encargalo/Encargalo.app-api.go/commit/e413912c50e33955d31e8418b73c6057c6e3925a))
* **trips:** impedir cancelar mientras el pago móvil está en revisión ([c1264a3](https://github.com/Encargalo/Encargalo.app-api.go/commit/c1264a392f88af624d5a03c89a9e8b22756e9297))
* **trips:** impedir cancelar mientras el pago móvil está en revisión ([b64f8b2](https://github.com/Encargalo/Encargalo.app-api.go/commit/b64f8b27f4b5efea956889b385f002f0ca65202a))
* **trips:** matching de trips a riders vía WebSocket (sub-PR 3/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([4e35f87](https://github.com/Encargalo/Encargalo.app-api.go/commit/4e35f872a320f4636e6a647ecc7c516c89fb0ed0))
* **trips:** matching de trips a riders vía WebSocket (sub-PR 3/4 de [#169](https://github.com/Encargalo/Encargalo.app-api.go/issues/169)) ([23638ce](https://github.com/Encargalo/Encargalo.app-api.go/commit/23638cec347bcb3576abc2bd34ffb7ed4c3b5015))
* **trips:** mostrar conductores cercanos en el mapa de cotización ([2e2fde6](https://github.com/Encargalo/Encargalo.app-api.go/commit/2e2fde657e66aa07dc392f5b7116bace43ac22a2))
* **trips:** mostrar conductores cercanos en el mapa de cotización ([886abae](https://github.com/Encargalo/Encargalo.app-api.go/commit/886abae977ff4a541b931e7b7845ffa496bd5e3b))
* **trips:** mostrar datos públicos del rider en el detalle y el historial ([67c9480](https://github.com/Encargalo/Encargalo.app-api.go/commit/67c94805a1d6602bf52dcb5dfa69e647f2b38a19))
* **trips:** mostrar datos públicos del rider en el detalle y el historial ([ecc6517](https://github.com/Encargalo/Encargalo.app-api.go/commit/ecc6517f9731f6a99b6ac0982f327e4bf0293076))
* **trips:** mostrar nombre y teléfono del pasajero al rider que acepta ([fbe2031](https://github.com/Encargalo/Encargalo.app-api.go/commit/fbe2031b239ab5a81304e15220e9a0f5cf7fa5f5))
* **trips:** mostrar nombre y teléfono del pasajero al rider que acepta ([6b3d540](https://github.com/Encargalo/Encargalo.app-api.go/commit/6b3d5400ad198dd2d87e186c83db736b55927650))
* **trips:** ofertar el viaje en rondas hasta agotar riders cercanos ([e34c132](https://github.com/Encargalo/Encargalo.app-api.go/commit/e34c13254591e8a501f4add3277b024d0640ffb8))
* **trips:** ofertar el viaje en rondas hasta agotar riders cercanos ([cd4b92e](https://github.com/Encargalo/Encargalo.app-api.go/commit/cd4b92e27431e8e24e459de9ed61868bffe430ac))
* **trips:** reofertar el viaje cuando el rider asignado cancela ([f3d0692](https://github.com/Encargalo/Encargalo.app-api.go/commit/f3d0692150a640275a26fb4ffe23dd56aa57a9cd))
* **trips:** reofertar el viaje cuando el rider asignado cancela ([540bb72](https://github.com/Encargalo/Encargalo.app-api.go/commit/540bb729a91ce7dc5df2fde70d639d41940e0cda))
* **trips:** seguir el viaje en vivo con códigos de subida/bajada y pago por saldo ([ee2d0ae](https://github.com/Encargalo/Encargalo.app-api.go/commit/ee2d0ae64c9b2ff98f30302b402f7ed1b4ab08eb))
* **trips:** seguir el viaje en vivo con códigos de subida/bajada y pago por saldo ([0cc2d56](https://github.com/Encargalo/Encargalo.app-api.go/commit/0cc2d56b373a2e37eb19ca507ab0d20832c54df7))
* **users:** devoluciones de balance, hold/processing/payout y monto Bs al CRM ([3dd8755](https://github.com/Encargalo/Encargalo.app-api.go/commit/3dd8755300c0624d18513c36cca60f8568e72709))
* **users:** eliminación de cuenta con período de gracia de 30 días ([d54776a](https://github.com/Encargalo/Encargalo.app-api.go/commit/d54776acf0955effe54f9468f733941bb5bb4b91))
* **users:** staff grant and revoke role endpoints ([a8d1888](https://github.com/Encargalo/Encargalo.app-api.go/commit/a8d1888eb4d9583b3b9e49986bf2dc3eefcaf56a))
* **users:** staff user management with cursor pagination ([4d7c58a](https://github.com/Encargalo/Encargalo.app-api.go/commit/4d7c58aba0e05dd2db41c156243a1f0f1f1b9288))


### Bug Fixes

* **analytics:** mapear PORT de Railway a SERVER_PORT en el ETL ([5ba7dae](https://github.com/Encargalo/Encargalo.app-api.go/commit/5ba7dae0426504da740d88f124dde1f95c72f4b4))
* **analytics:** partir upserts OLAP bajo el límite de parámetros de Postgres ([382b6a6](https://github.com/Encargalo/Encargalo.app-api.go/commit/382b6a6b1e370aa9c4845b926d216aa358cc1a4b))
* **auth:** rate limiting, ip/user-agent y manejo de errores en sign-in ([f3754e1](https://github.com/Encargalo/Encargalo.app-api.go/commit/f3754e1b50a8195c8e7da73f4f5a19a7c75d7c5c))
* **auth:** registrar /internal/verify-session como GET, no POST ([e17a18e](https://github.com/Encargalo/Encargalo.app-api.go/commit/e17a18e9403ed34cf904f085b6464e5e8e24a110))
* **auth:** responder logout con un objeto json en vez de string plano ([d7e84f0](https://github.com/Encargalo/Encargalo.app-api.go/commit/d7e84f09ce52b8c782c511484ed86fe284a8823b))
* calificar columnas en relación Item.Discount para evitar ambigüe… ([3c98120](https://github.com/Encargalo/Encargalo.app-api.go/commit/3c98120bf62c766e31a3e29f4f9002c734ae87e2))
* calificar columnas en relación Item.Discount para evitar ambigüedad SQL ([be9f405](https://github.com/Encargalo/Encargalo.app-api.go/commit/be9f405e9e23dcae425a3bdba9d5be50ca822096))
* **ci:** correr coverage tras merge a main y detectar tests con find ([2f6d45a](https://github.com/Encargalo/Encargalo.app-api.go/commit/2f6d45ae1616f37a9237d4a0cfbe92f7672c2869))
* **ci:** correr coverage tras merge a main y detectar tests con find ([4ae456e](https://github.com/Encargalo/Encargalo.app-api.go/commit/4ae456e7b0a9bc123264fb90befd73d98b195f98))
* **ci:** medir cobertura real con -coverpkg=./... ([5844b5a](https://github.com/Encargalo/Encargalo.app-api.go/commit/5844b5a6f020406525c8737c731db4885aa0d150))
* **ci:** medir cobertura real con -coverpkg=./... ([e5d04a9](https://github.com/Encargalo/Encargalo.app-api.go/commit/e5d04a9af1164cdcf328df0da675cf60880a4615))
* **ci:** reparar badges de codecov y release en README ([38ec6c2](https://github.com/Encargalo/Encargalo.app-api.go/commit/38ec6c284d7cf610d83feb50a2826ce11ee92dc1))
* **ci:** reparar badges de codecov y release en README ([7251fc1](https://github.com/Encargalo/Encargalo.app-api.go/commit/7251fc193e23bb995e1e5d32269f4c1ce1e57499))
* **config:** reconocer produccion de Railway y ocultar items hidden en trending ([e46bc8f](https://github.com/Encargalo/Encargalo.app-api.go/commit/e46bc8f17fbc3df552e2e8f7b8ffaeb1284a26cb))
* corregir bugs en lógica BCV y caché de precios en productos ([04363d5](https://github.com/Encargalo/Encargalo.app-api.go/commit/04363d560f750fbdb249d16d6f9c6d9ad7089825))
* **deps:** actualizar echo/v4 a v4.15.3 por CVE-2026-55677 ([946e3e9](https://github.com/Encargalo/Encargalo.app-api.go/commit/946e3e900d165a8c12b26a130e620e31d172900b))
* **deps:** actualizar grpc a v1.82.1 y eliminar integración Slack ([e6f7d84](https://github.com/Encargalo/Encargalo.app-api.go/commit/e6f7d84bdf8bac2aae174fdb26678eaaeee2a709))
* **deps:** cerrar alertas de Dependabot en grpc y OpenTelemetry ([e310e4a](https://github.com/Encargalo/Encargalo.app-api.go/commit/e310e4aa7e52727c5a266ea8db84261d09550288))
* **deps:** cerrar alertas de Dependabot en grpc y OpenTelemetry ([106b201](https://github.com/Encargalo/Encargalo.app-api.go/commit/106b2015bea39de70637d891d080fd4140775f9b))
* **diagnostics:** requerir sesión con rol Staff para /debug/pprof ([b22759e](https://github.com/Encargalo/Encargalo.app-api.go/commit/b22759e4a65a3c5e941bcb10e33cad66e7e711ba))
* embed tzdata en el binario para que cargue America/Caracas en ru… ([c25e97e](https://github.com/Encargalo/Encargalo.app-api.go/commit/c25e97e2b7b1afbfcc54207a20aa320865b86781))
* embed tzdata en el binario para que cargue America/Caracas en runtime ([54a64a7](https://github.com/Encargalo/Encargalo.app-api.go/commit/54a64a7adafa12aa06ba6ea436f989ec0d6d21ea))
* **migrations:** corregir columnas del INSERT en payment_methods ([5dbe06d](https://github.com/Encargalo/Encargalo.app-api.go/commit/5dbe06d4f0ad52dc6ece25e4cb57afe240c60980))
* **observability:** forzar el path OTLP correcto por señal ([57e225c](https://github.com/Encargalo/Encargalo.app-api.go/commit/57e225cb0e074aac9056df6d7ff3d99750d9e2a6))
* **observability:** incluir método, ruta y status en el mensaje de logs HTTP ([0b8e708](https://github.com/Encargalo/Encargalo.app-api.go/commit/0b8e7084a7f6ea0e13f83f5080341e5799c70d88))
* **observability:** usar el patrón de ruta en vez de la URL cruda en el campo endpoint ([9a98a99](https://github.com/Encargalo/Encargalo.app-api.go/commit/9a98a99653f7fc8ba0f3230600db1057aa64f48b))
* **orders,products:** calificar columnas ambiguas tras el JOIN a discounts ([935ba2d](https://github.com/Encargalo/Encargalo.app-api.go/commit/935ba2dcefbcf30340b24051d9ae1c6c0e17e403))
* **orders:** corregir BOLA cross-shop en ciclo de vida de órdenes ([c76f360](https://github.com/Encargalo/Encargalo.app-api.go/commit/c76f360e9d02676ff98746f0f761add1ca820cd4))
* **orders:** guardar created_at real al crear una orden ([07ea8c3](https://github.com/Encargalo/Encargalo.app-api.go/commit/07ea8c37a7403d62f39552ac2295795899bd6ca4))
* **orders:** incrementar sold del producto en async sin pasar por redis ([ee5a030](https://github.com/Encargalo/Encargalo.app-api.go/commit/ee5a0304e303a4da6b8d8e2eaa66da6e2f95aecc))
* **orders:** limitar tamano del comprobante de pago movil ([aca77fe](https://github.com/Encargalo/Encargalo.app-api.go/commit/aca77fede62866285b7ad4cf57ab51e93a2291c0))
* **orders:** no consumir cotización si falla el comprobante ([cd17a18](https://github.com/Encargalo/Encargalo.app-api.go/commit/cd17a18bb8277d5017234c16623e3c38d7687bcb))
* **orders:** romper ciclo de DI usando ProductsRepo en lugar de ProductsApp ([48ada1e](https://github.com/Encargalo/Encargalo.app-api.go/commit/48ada1e08af69118a660cc4575b0c5e8236314ed))
* **orders:** usar el mismo delivery fee placeholder que el listado fuera de production ([9ea38b6](https://github.com/Encargalo/Encargalo.app-api.go/commit/9ea38b61910a852ec9c8e5be1644c0ba90df8716))
* **products:** corregir alias del JOIN en relación Item.Discount ([d7369aa](https://github.com/Encargalo/Encargalo.app-api.go/commit/d7369aaf9603bb634a1cd83ac0a68e9bc3296bd4))
* **server:** usar X-Real-Ip para c.RealIP() en vez de X-Forwarded-For ([21580a9](https://github.com/Encargalo/Encargalo.app-api.go/commit/21580a9e9eea6a2b7c93d742648763ce7a1be877))
* **shops:** count only sale movements in stats total earned ([402a5b5](https://github.com/Encargalo/Encargalo.app-api.go/commit/402a5b59c916d24cae2c8a22079aea6062eadd24))
* **shops:** usar fecha de transacción para stats de órdenes completadas ([3356851](https://github.com/Encargalo/Encargalo.app-api.go/commit/335685122fb331b7bf6c9f684b421aadd4dc428d))
* **shops:** usar fecha de transacción para stats de órdenes completadas ([db4e405](https://github.com/Encargalo/Encargalo.app-api.go/commit/db4e4050c54fdedbf685631f481e2f1131244482))
* **trips:** corregir FK de customer_id a users.accounts ([fce0e5c](https://github.com/Encargalo/Encargalo.app-api.go/commit/fce0e5cec850cfb68248bf951bc157394d1ceb7b))
* **trips:** corregir FK de customer_id a users.accounts ([93c6980](https://github.com/Encargalo/Encargalo.app-api.go/commit/93c69808bad6c64dad5f1bbc86e7dc273a0ac87e))
* **trips:** mitigar spam de advance_trip y generalizar mensaje de transición ([64e2793](https://github.com/Encargalo/Encargalo.app-api.go/commit/64e2793fd1b518aaa9023ec61da412cfceba727e))
* **trips:** propaga fallos de transición y evita filtrar el modelo de dominio por WebSocket ([bdb89d2](https://github.com/Encargalo/Encargalo.app-api.go/commit/bdb89d2e39fe8d7c99ff2e5ee69acf3e11706e5a))
* **trips:** rondas de búsqueda de 10 min y rondas vacías de 5 min ([193f64a](https://github.com/Encargalo/Encargalo.app-api.go/commit/193f64af5234af39935432c18b8c6f44b17b5722))
* **trips:** rondas de búsqueda de 10 min y rondas vacías de 5 min ([f257a11](https://github.com/Encargalo/Encargalo.app-api.go/commit/f257a11eba4bd9a5c86c88f200e40bc50eddb5f9))
* **trips:** rondas de búsqueda de 15 min y esperar también en rondas vacías ([cedfc3f](https://github.com/Encargalo/Encargalo.app-api.go/commit/cedfc3f28ab092a1de19b7b7f6385a0a1f193829))
* **users:** errores crudos, sesión inválida en registro y código muerto ([6b9968c](https://github.com/Encargalo/Encargalo.app-api.go/commit/6b9968c32858efe38d30db1b475e48fe127c73c4))
* **users:** no guardar email vacío en el registro de customers ([b7e9574](https://github.com/Encargalo/Encargalo.app-api.go/commit/b7e9574e0be2ec9bd9844caa4d2a7fdc7663414c))


### Refactors

* **analytics:** usar config mínima dedicada para cmd/etl ([2d0c87f](https://github.com/Encargalo/Encargalo.app-api.go/commit/2d0c87f6800dda52303fbd781e35abee3af2a975))
* **finance:** balance y movimientos solo en balance_bs/balance_usd ([37a62f6](https://github.com/Encargalo/Encargalo.app-api.go/commit/37a62f6cfda4ce0e75793f3fea40fb35abae1615))
* **handlers:** simplificar acceso a sesión removiendo verificación redundante ([a4d50a9](https://github.com/Encargalo/Encargalo.app-api.go/commit/a4d50a9925513420adda8db46ddcc2fc250bb2a7))
* **orders:** aplicar strategy pattern en UpdateOrderStatus y corregir violaciones arquitectónicas ([e7e5d4a](https://github.com/Encargalo/Encargalo.app-api.go/commit/e7e5d4a191975365e8ae1a7dd3544bdad3aa478e))
* **orders:** strategy pattern en UpdateOrderStatus + correcciones arquitectónicas ([b37c0cd](https://github.com/Encargalo/Encargalo.app-api.go/commit/b37c0cd15cb1c07a64b071d03c13d35f544e8ce6))
* **riders:** corregir violaciones arquitectónicas y agregar tests ([3c2ce2b](https://github.com/Encargalo/Encargalo.app-api.go/commit/3c2ce2bd2eda25ec0edb5d4a539a5a11678cd62b))
* **shops:** unificar cálculo de comisión del negocio ([58586f4](https://github.com/Encargalo/Encargalo.app-api.go/commit/58586f439fcc1b62653512838eff543e431cc675))
* **shops:** unificar cálculo de comisión del negocio ([d896b14](https://github.com/Encargalo/Encargalo.app-api.go/commit/d896b14a9d54da5aae8e708f6d7c8f1e9256ff23))


### Performance

* **postgres:** configurar pool de conexiones y evitar reconexión por query ([784c6d9](https://github.com/Encargalo/Encargalo.app-api.go/commit/784c6d9f15258e70730b86208a74539793ed599f))
* **riders:** cancelar el timer de expiración de trip al aceptarlo ([784c6d9](https://github.com/Encargalo/Encargalo.app-api.go/commit/784c6d9f15258e70730b86208a74539793ed599f))
* **riders:** quitar COUNT(*) innecesario del preview de saldo ([cc74c43](https://github.com/Encargalo/Encargalo.app-api.go/commit/cc74c432be49ed1ce37ae1ba7699d9bb771ceec6))
* **trips:** colapsar lectura + update de AdvanceTrip en una sola query ([43148f6](https://github.com/Encargalo/Encargalo.app-api.go/commit/43148f6fb75f2ec8e74006d91d70ccfb374242bc))
* **trips:** eliminar índices sin uso en trips.trips ([7d61fac](https://github.com/Encargalo/Encargalo.app-api.go/commit/7d61fac88edfb3ed429635362dfed6230fb87c96))


### Documentation

* agregar auditoría de arquitectura y seguridad ([3f2b72b](https://github.com/Encargalo/Encargalo.app-api.go/commit/3f2b72b0f74a204f9c089f499eba08ddfbb432bf))
* **postman:** limpiar colecciones y agregar environments por entorno ([55d0e1e](https://github.com/Encargalo/Encargalo.app-api.go/commit/55d0e1e3022ac94d08e0789effa4074fa8f010ad))
* prohibir firma de IA en commits y PRs ([c9c189b](https://github.com/Encargalo/Encargalo.app-api.go/commit/c9c189bdd8195a63ba07fc48d0da9d184bed6592))
* **riders:** regenerar swagger con GET /riders/trips/active ([f4ba742](https://github.com/Encargalo/Encargalo.app-api.go/commit/f4ba7421e3fa925b09e0b20e0028d9ec7769f91a))

## [1.9.0](https://github.com/Encargalo/Encargalo.app-api.go/compare/v1.8.0...v1.9.0) (2026-05-29)


### Features

* add payment methods domain and update order creation flow ([9704ac8](https://github.com/Encargalo/Encargalo.app-api.go/commit/9704ac8e4406481c7618d9da3af40eacb0961b52))
* add payment methods domain and update order creation flow ([1fe30a5](https://github.com/Encargalo/Encargalo.app-api.go/commit/1fe30a51eec7ff5cb52c197565d1fded51c9956d))

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
