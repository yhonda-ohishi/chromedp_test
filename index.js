import { Container, getContainer } from "@cloudflare/containers";

export class MyContainer extends Container {
    defaultPort = 8080; // Port the container is listening on
    sleepAfter = "45m"; // Stop the instance if requests not sent for 45 minutes (延長)
    maxInstances = 1; // 同時実行インスタンス数を制限
    memoryMB = 2048; // メモリ使用量を増加
    cpuMs = 10000; // CPU時間を大幅に延長
}

export default {
    async fetch(request, env) {
        try {
            const containerInstance = getContainer(env.MY_CONTAINER, "test");

            // タイムアウト設定付きでリクエストを実行
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 600000); // 10分タイムアウト

            const response = await containerInstance.fetch(request, {
                signal: controller.signal
            });

            clearTimeout(timeoutId);
            return response;

        } catch (error) {
            console.error("Container request failed:", error);
            return new Response(JSON.stringify({
                status: "error",
                message: "サービスが一時的に利用できません。しばらく後でもう一度お試しください。",
                error: error.message
            }), {
                status: 503,
                headers: { "Content-Type": "application/json" }
            });
        }
    }
}