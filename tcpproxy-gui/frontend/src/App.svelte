<script lang="ts">
  import {
    Button,
    Card,
    Checkbox,
    Input,
    Label,
    Modal,
    Radio,
    Toast,
  } from "flowbite-svelte";

  import TaskPage from "./TaskPage.svelte";
  import { onMount } from "svelte";
  import { GetAllTask, Quit } from "./lib/wailsjs/go/main/App.js";
  import { Icon } from "svelte-awesome";
  import {
    faCircleInfo,
    faInfo,
    faPlus,
    faWarning,
  } from "@fortawesome/free-solid-svg-icons";
  import { EventsOn, Hide } from "./lib/wailsjs/runtime/runtime";

  let datas = [];
  let tmpTitle = "";
  let showNewModule = false;
  let showQueryQuit = false;
  let toastData = [];
  let quitFlag = "tray";
  let quitOpt = false;

  onMount(() => {
    GetAllTask().then((res) => (datas = res));
  });
  function AddTask() {
    if (!tmpTitle) {
      return;
    }
    datas.push({
      title: tmpTitle,
      host: "127.0.0.1",
      ports: [{ from: 1, to: 1 }],
    });
    tmpTitle = "";
    datas = datas;
  }
  function AddToast(msg: string, icon, color) {
    toastData.push({ msg, icon, color });
    toastData = toastData;
  }
  EventsOn("toast", AddToast);
  function QueryQuit() {
    if (quitOpt) {
      if (quitFlag == "quit") {
        Quit();
      } else {
        Hide();
        showQueryQuit = false;
      }
    } else {
      showQueryQuit = true;
    }
  }
  function BtnQuitOpt() {
    if (quitFlag == "quit") {
      Quit();
    } else {
      Hide();
    }
  }
  EventsOn("queryQuit", QueryQuit);
</script>

<main class="flex flex-wrap">
  {#each toastData as t}
    <Toast color={t.color ?? "green"} position="top-right">
      <svelte:fragment slot="icon">
        {#if t.icon}
          <Icon data={t.icon} />
        {:else}
          <Icon data={faCircleInfo} />
        {/if}
      </svelte:fragment>
      {t.msg}
    </Toast>
  {/each}
  {#each datas as data, index}
    <TaskPage open={index == 0} {...data} class="w-full" />
  {/each}
  <Card>
    <Button on:click={() => (showNewModule = true)}>
      <Icon data={faPlus} scale={2} /> 新加配置
    </Button>
  </Card>
  <Modal bind:open={showNewModule} size="xs" autoclose class="w-full">
    <Label class="space-y-2">
      <span>请输入新服务器名称</span>
      <Input name="title" bind:value={tmpTitle} />
    </Label>
    <svelte:fragment slot="footer">
      <div class="space-x-2">
        <Button color="red" on:click={AddTask}>确定</Button>
        <Button color="blue" on:click={() => (showNewModule = false)}
          >取消</Button
        >
      </div>
    </svelte:fragment>
  </Modal>
  <Modal bind:open={showQueryQuit} size="xs" autoclose>
    <h1 class="text-3xl font-bold">关闭提示</h1>
    <Radio bind:group={quitFlag} value="tray">最小化到系统托盘</Radio>
    <Radio bind:group={quitFlag} value="quit">退出协议代理工具</Radio>
    <Button on:click={BtnQuitOpt} color="red">确定</Button>
    <Checkbox bind:checked={quitOpt}>不再提醒</Checkbox>
  </Modal>
</main>

<style>
</style>
