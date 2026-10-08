# Simple API (SAS)

Ferramenta de linha de comando pequena para fazer requisições HTTP durante o
desenvolvimento e os testes manuais de APIs. Escrita em Go, sem dependências
externas, funciona em Linux, macOS e Windows.

O objetivo é reduzir a repetição de comandos `curl`, permitir reutilizar
configurações por variáveis de ambiente e manter uma implementação simples,
legível e multiplataforma.

## Instalação

```bash
go install github.com/ShiMigui/simple-api-spec/cmd/sas@latest
```

Ou, a partir do código:

```bash
go build -o sas ./cmd/sas
```

## Uso

Dois formatos equivalentes, ambos sobre o mesmo núcleo:

```text
sas METHOD URL [opções]
sas-METHOD URL [opções]
```

Exemplos:

```bash
sas get /health
sas post /users -t application/json -b '{"name":"Ana"}'
sas patch /users/42 -t application/json -b '{"active":false}'
sas delete /users/42

# Mesmo fluxo através do nome do executável (symlink ou cópia no Windows):
sas-post /users -t application/json -b '{"name":"Ana"}'
```

Métodos do MVP: `GET`, `POST`, `PUT`, `PATCH` e `DELETE`.

### Opções

```text
-b, --body VALUE            corpo literal da requisição
-H, --header "Name: value"  header adicional; pode ser repetido
-t, --content-type TYPE     Content-Type do corpo
-h, --help                  ajuda
    --version               versão
```

O corpo informado em `-b` é tratado como bytes literais: nunca é interpretado,
corrigido ou reformatado como JSON.

## Variáveis de ambiente

| Variável            | Uso                                                  |
| ------------------- | ---------------------------------------------------- |
| `API_BASE_URL`      | URL base para endpoints relativos                    |
| `API_BEARER_TOKEN`  | enviado como `Authorization: Bearer <token>`         |
| `API_CONTENT_TYPE`  | `Content-Type` padrão                                |
| `API_ACCEPT`        | valor padrão de `Accept`                             |
| `API_TIMEOUT`       | timeout da requisição, por exemplo `30s` (padrão 30s)|
| `API_VERBOSE`       | ativa logs detalhados                                |

Toda a leitura de ambiente fica isolada em `internal/config`; nenhum outro
pacote chama `os.Getenv`.

### Precedência

Resolvida em um único ponto (`config.Resolve`):

```text
opção de CLI  >  variável de ambiente  >  valor padrão
```

Regras específicas:

- `-t` prevalece sobre `API_CONTENT_TYPE`.
- Um header explícito `-H` prevalece sobre valores automáticos
  (`API_ACCEPT`, `API_BEARER_TOKEN` e `Content-Type` padrão).
- Usar `-t` junto de `-H 'Content-Type: ...'` é rejeitado como ambíguo.

## Resolução de URL

- URLs absolutas `http://` e `https://` são usadas diretamente.
- Endpoints relativos exigem `API_BASE_URL`; sua ausência é um erro claro.
- O endpoint é **anexado** ao caminho base (comportamento explícito, diferente
  da resolução tradicional que substituiria o caminho).
- Barras inicial/final são normalizadas; o caminho-base é preservado.
- Query strings são preservadas; fragmentos não são enviados.

Exemplos:

```text
base https://host/api/v1  +  /users          -> https://host/api/v1/users
base http://host:8080/api/ +  users          -> http://host:8080/api/users
URL absoluta https://example.com/api/users   -> usada como está
```

## Saída e códigos de retorno

- O **corpo** da resposta vai para `stdout`, sem formatação automática.
- Diagnósticos e erros vão para `stderr`.

Assim, o redirecionamento funciona sem poluição:

```bash
sas get /users > users.json   # users.json contém apenas o corpo
```

Códigos de saída:

| Código | Significado                                  |
| ------ | -------------------------------------------- |
| `0`    | resposta HTTP de sucesso (`2xx`)             |
| `1`    | erro de transporte ou de configuração        |
| `2`    | argumentos/opções inválidos                  |
| `3`    | resposta HTTP não bem-sucedida (`4xx`/`5xx`) |

Em respostas não bem-sucedidas o corpo ainda é escrito em `stdout` e o status
é reportado em `stderr`.

## Segurança

- Tokens e credenciais nunca são impressos em logs, erros ou no modo verbose.
- Headers são validados (`Nome: valor`); CR/LF e caracteres inválidos são
  rejeitados para evitar injeção.
- A validação TLS permanece ativa.
- Uma requisição é redirecionada no máximo 10 vezes; credenciais não são
  propagadas para hosts diferentes em redirecionamentos.

## Estrutura do projeto

```text
cmd/sas/main.go          ponto de entrada; converte o resultado em código de saída
internal/apperr          classificação de erros -> códigos de saída
internal/cli             parsing de argumentos (um único Options para todos os formatos)
internal/config          leitura e resolução de ambiente (env.go, config.go)
internal/request         resolução de URL, headers, corpo e cliente HTTP
internal/response        escrita do corpo e diagnósticos nos streams corretos
```

Todos os métodos e formatos de invocação convergem para o mesmo fluxo HTTP.

## Testes

```bash
go test ./...
go test -race ./...
```

Os testes usam `net/http/httptest`, sem dependência de APIs externas, e cobrem
resolução de URL, precedência de configuração, headers, corpo, timeout,
redirecionamentos, separação de streams e códigos de saída.
