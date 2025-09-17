import os, sys, json, time, base64
import click
import pika
from prometheus_client import Counter, Histogram, start_http_server

REPLAYED = Counter('dlqctl_replayed_total','DLQ messages replayed', ['queue'])
QUARANTINED = Counter('dlqctl_quarantined_total','DLQ messages quarantined', ['queue','reason'])
ANALYZED = Counter('dlqctl_analyzed_total','DLQ messages analyzed', ['queue'])
LATENCY = Histogram('dlqctl_op_latency_seconds','DLQ op latency', ['op'])

DEFAULT_DLQS = ['shipments.dlq','labels.dlq','pricing.dlq','tracking.dlq']

class RMQ:
    def __init__(self, url):
        self.conn = pika.BlockingConnection(pika.URLParameters(url))
        self.ch = self.conn.channel()

    def close(self):
        try: self.conn.close()
        except: pass

    def declare(self, q):
        self.ch.queue_declare(queue=q, durable=True)

    def get(self, q, ack=False):
        return self.ch.basic_get(queue=q, auto_ack=not ack)

    def publish(self, q, body, headers=None):
        props = pika.BasicProperties(content_type='application/json', headers=headers or {})
        self.ch.basic_publish(exchange='', routing_key=q, body=body, properties=props)

@click.group()
@click.option('--rmq-url', envvar='RABBITMQ_URL', default='amqp://guest:guest@rabbitmq:5672/', show_default=True)
@click.option('--metrics-port', envvar='DLQCTL_METRICS_PORT', default=9109, show_default=True, type=int)
@click.pass_context
def cli(ctx, rmq_url, metrics_port):
    start_http_server(metrics_port)
    ctx.ensure_object(dict)
    ctx.obj['rmq'] = RMQ(rmq_url)

@cli.command()
@click.option('--dlq', multiple=True, help='DLQ name(s) to analyze')
@click.pass_context
def analyze(ctx, dlq):
    dlqs = list(dlq) or DEFAULT_DLQS
    rmq = ctx.obj['rmq']
    with LATENCY.labels('analyze').time():
        for q in dlqs:
            rmq.declare(q)
            msg = rmq.get(q, ack=False)
            if not msg[0]:
                click.echo(f"{q}: empty")
                continue
            method, props, body = msg
            ANALYZED.labels(q).inc()
            headers = props.headers or {}
            xdeath = headers.get('x-death')
            schema_ver = headers.get('x-schema-version')
            click.echo(json.dumps({
                'queue': q,
                'schema_version': schema_ver,
                'x_death': xdeath,
                'content_type': props.content_type,
                'size': len(body)
            }))

@cli.command()
@click.option('--dlq', multiple=True, help='DLQ name(s) to replay')
@click.option('--max', 'limit', default=100, show_default=True, help='Max messages per queue to process')
@click.option('--poison-threshold', default=3, show_default=True, help='x-death count threshold to quarantine')
@click.option('--dry-run', is_flag=True, help='Analyze only, no mutations')
@click.pass_context
def replay(ctx, dlq, limit, poison_threshold, dry_run):
    dlqs = list(dlq) or DEFAULT_DLQS
    rmq = ctx.obj['rmq']
    with LATENCY.labels('replay').time():
        for q in dlqs:
            rmq.declare(q)
            base = q.replace('.dlq','')
            orig = base
            poison = f"{base}.poison"
            rmq.declare(orig); rmq.declare(poison)
            count = 0
            while count < limit:
                method, props, body = rmq.get(q, ack=True)
                if not method: break
                headers = (props.headers or {}).copy()
                xdeath = headers.get('x-death')
                deaths = 0
                if isinstance(xdeath, list) and xdeath:
                    deaths = xdeath[0].get('count', 0)
                if deaths >= poison_threshold:
                    if dry_run:
                        click.echo(f"DRYRUN quarantine {q} -> {poison} (deaths={deaths})")
                    else:
                        rmq.publish(poison, body, headers=headers)
                        QUARANTINED.labels(q,'x-death-threshold').inc()
                    # ack consumed message (auto_ack=False -> we consumed; pika basic_get with auto_ack=False means we must ack, but we used ack=True for manual ack)
                    ctx.obj['rmq'].ch.basic_ack(method.delivery_tag)
                else:
                    if dry_run:
                        click.echo(f"DRYRUN replay {q} -> {orig}")
                    else:
                        rmq.publish(orig, body, headers=headers)
                        REPLAYED.labels(q).inc()
                        ctx.obj['rmq'].ch.basic_ack(method.delivery_tag)
                count += 1
            click.echo(f"{q}: processed={count}")

@cli.command()
@click.option('--dlq', default='shipments.dlq')
@click.option('--limit', default=10)
@click.pass_context
def peek(ctx, dlq, limit):
    rmq = ctx.obj['rmq']
    rmq.declare(dlq)
    n=0
    with LATENCY.labels('peek').time():
        while n < limit:
            method, props, body = rmq.get(dlq, ack=False)
            if not method: break
            headers = props.headers or {}
            click.echo(json.dumps({'headers': headers, 'body_b64': base64.b64encode(body).decode()}))
            n+=1

if __name__ == '__main__':
    try:
        cli(obj={})
    except KeyboardInterrupt:
        pass

