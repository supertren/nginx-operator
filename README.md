# nginx-operator

Operador OpenShift desarrollado con operator-sdk y Go que gestiona un Deployment de nginx con N réplicas mediante un Custom Resource `NginxCluster`.

## ¿Qué hace este operador?

Cuando creas un objeto `NginxCluster` en OpenShift, el operador automáticamente:

1. Crea un `Deployment` de nginx con las réplicas especificadas
2. Crea un `Service` apuntando a los pods nginx en el puerto 8080
3. Actualiza el campo `status.availableReplicas` del CR con el estado real
4. Si alguien borra el Deployment manualmente, el operador lo recrea automáticamente (owner references)

## Prerrequisitos

En tu máquina necesitas:

- **Windows 11** con Hyper-V habilitado
- **CRC (OpenShift Local)** instalado y configurado (mínimo 8 vCPUs, 16GB RAM, 60GB disco)
- **WSL2** con Ubuntu y git instalados
- **PowerShell** (sin privilegios de administrador)
- **oc CLI** en el PATH de PowerShell (incluido con CRC)

## Instalación paso a paso

### PASO 1 — Arrancar CRC (PowerShell)

```powershell
crc start
& crc oc-env | Invoke-Expression
oc login -u kubeadmin https://api.crc.testing:6443
```

Verifica que el cluster está operativo:

```powershell
oc get nodes
# NAME   STATUS   ROLES                         AGE   VERSION
# crc    Ready    control-plane,master,worker   Xd    v1.34.x
```

### PASO 2 — Clonar el repositorio (WSL2)

```bash
cd ~
git clone https://github.com/supertren/nginx-operator.git
cd nginx-operator
```

### PASO 3 — Crear el namespace y el BuildConfig (PowerShell)

```powershell
oc new-project nginx-operator
oc new-build --name=nginx-operator --binary --strategy=docker -n nginx-operator
```

### PASO 4 — Build de la imagen del operador (PowerShell)

Sustituye `TU_USUARIO` por tu usuario de WSL2:

```powershell
oc start-build nginx-operator --from-dir="\\wsl$\Ubuntu\home\TU_USUARIO\nginx-operator" --follow -n nginx-operator
```

Al finalizar verás:

```
Push successful
```

### PASO 5 — Instalar el CRD (PowerShell)

Sustituye `TU_USUARIO` por tu usuario de WSL2:

```powershell
oc apply -f "\\wsl$\Ubuntu\home\TU_USUARIO\nginx-operator\config\crd\bases\apps.example.com_nginxclusters.yaml" -n nginx-operator
```

### PASO 6 — Configurar el RBAC (PowerShell)

```powershell
oc apply -f "\\wsl$\Ubuntu\home\TU_USUARIO\nginx-operator\config\rbac\" -n nginx-operator
oc create serviceaccount controller-manager -n nginx-operator
oc adm policy add-cluster-role-to-user manager-role -z controller-manager -n nginx-operator
```

> Los errores `namespace "system"` y `kustomization.yaml` son esperados e irrelevantes. El RBAC se aplica correctamente.

### PASO 7 — Desplegar el operador (PowerShell)

```powershell
$yaml = "apiVersion: apps/v1`nkind: Deployment`nmetadata:`n  name: nginx-operator`n  namespace: nginx-operator`nspec:`n  replicas: 1`n  selector:`n    matchLabels:`n      app: nginx-operator`n  template:`n    metadata:`n      labels:`n        app: nginx-operator`n    spec:`n      serviceAccountName: controller-manager`n      containers:`n      - name: manager`n        image: image-registry.openshift-image-registry.svc:5000/nginx-operator/nginx-operator:latest`n        command:`n        - /manager"
$yaml | Out-File -FilePath "C:\Users\$env:USERNAME\nginx-operator-deployment.yaml" -Encoding utf8
oc apply -f "C:\Users\$env:USERNAME\nginx-operator-deployment.yaml"
```

Verifica que el pod del operador está Running:

```powershell
oc get pods -n nginx-operator
# NAME                              READY   STATUS    RESTARTS   AGE
# nginx-operator-xxxxxxxxx-xxxxx   1/1     Running   0          30s
```

### PASO 8 — Crear el Custom Resource NginxCluster (PowerShell)

```powershell
$yaml = "apiVersion: apps.example.com/v1alpha1`nkind: NginxCluster`nmetadata:`n  name: mi-nginx`n  namespace: nginx-operator`nspec:`n  replicas: 3"
$yaml | Out-File -FilePath "C:\Users\$env:USERNAME\nginxcluster-test.yaml" -Encoding utf8
oc apply -f "C:\Users\$env:USERNAME\nginxcluster-test.yaml"
```

### PASO 9 — Verificar el resultado (PowerShell)

```powershell
oc get pods -n nginx-operator
```

Resultado esperado:

```
NAME                              READY   STATUS    RESTARTS   AGE
mi-nginx-xxxxxxxxx-xxxxx         1/1     Running   0          30s
mi-nginx-xxxxxxxxx-xxxxx         1/1     Running   0          30s
mi-nginx-xxxxxxxxx-xxxxx         1/1     Running   0          30s
nginx-operator-xxxxxxxxx-xxxxx   1/1     Running   0          2m
```

Verifica el estado del CR:

```powershell
oc get nginxcluster mi-nginx -n nginx-operator -o yaml
# status:
#   availableReplicas: 3
```

Verifica los logs del operador:

```powershell
oc logs deployment/nginx-operator -n nginx-operator --tail=10
```

## Probar owner references

Si borras el Deployment de nginx manualmente, el operador lo recrea automáticamente:

```powershell
oc delete deployment mi-nginx -n nginx-operator
# Espera 5 segundos
oc get pods -n nginx-operator
# Los 3 pods de nginx vuelven a aparecer
```

## Cambiar el número de réplicas

```powershell
$yaml = "apiVersion: apps.example.com/v1alpha1`nkind: NginxCluster`nmetadata:`n  name: mi-nginx`n  namespace: nginx-operator`nspec:`n  replicas: 5"
$yaml | Out-File -FilePath "C:\Users\$env:USERNAME\nginxcluster-test.yaml" -Encoding utf8
oc apply -f "C:\Users\$env:USERNAME\nginxcluster-test.yaml"
oc get pods -n nginx-operator
```

El operador detecta el cambio y escala el Deployment automáticamente.

## Limpiar el entorno

```powershell
oc delete project nginx-operator
```

## Estructura del proyecto

```
nginx-operator/
├── api/v1alpha1/
│   └── nginxcluster_types.go      # Definición del CRD (Spec + Status)
├── internal/controller/
│   └── nginxcluster_controller.go # Lógica del reconciliation loop
├── config/
│   ├── crd/bases/                 # YAML del CRD generado
│   └── rbac/                      # ServiceAccount, ClusterRole, Bindings
├── cmd/main.go                    # Punto de entrada del operador
└── Dockerfile                     # Multi-stage build (golang:1.24 + distroless)
```

## Notas técnicas

- **Imagen nginx**: se usa `nginxinc/nginx-unprivileged:latest` en lugar de `nginx:latest` porque OpenShift aplica SCC `restricted-v2` que prohíbe contenedores como root
- **Puerto**: 8080 (nginx-unprivileged no puede bindear puertos menores de 1024)
- **Build**: ocurre dentro del clúster CRC usando `BuildConfig` con input binario, sin necesidad de registry externo
- **WSL2 ↔ PowerShell**: el código reside en WSL2 y se envía al clúster mediante la ruta UNC `\\wsl$\Ubuntu\...`
