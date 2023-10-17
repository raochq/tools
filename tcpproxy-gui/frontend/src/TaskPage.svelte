<script lang="ts">
    import {
        Button,
        ButtonGroup,
        Card,
        Input,
        Label,
        Table,
        TableBody,
        TableBodyCell,
        TableBodyRow,
        TableHead,
        TableHeadCell,
    } from "flowbite-svelte";

    import Icon from "svelte-awesome";
    import {
        faMinusCircle,
        faPlay,
        faPlusCircle,
        faSave,
        faStop,
        faTimesCircle,
        faWarning,
        faXmarkCircle,
    } from "@fortawesome/free-solid-svg-icons";

    import { StartTask, StopTask } from "./lib/wailsjs/go/main/App";
    import { onMount } from "svelte";
    import { IsTaskRun } from "./lib/wailsjs/go/main/App";
    import { SaveTask } from "./lib/wailsjs/go/main/App";
    import type { main } from "./lib/wailsjs/go/models";
    import { DelTask } from "./lib/wailsjs/go/main/App";
    import { EventsEmit } from "./lib/wailsjs/runtime/runtime";

    export let title = "";
    export let host = "";
    export let ports = [];
    let ref;
    let working = false;
    function DeleteRecord(i) {
        if (working) {
            alert(`${title} is running`);
            return;
        }
        if (i >= 0 && i < ports.length) {
            ports.splice(i, 1);
            ports = ports;
        }
    }
    function AddRecord() {
        ports.push({ from: 0, to: 0 });
        ports = ports;
    }
    function doWrok() {
        if (working) {
            StopTask(title).then(() => (working = false));
        } else {
            StartTask(title, host, checkPors())
                .then((res) => (working = true))
                .catch((err) => {
                    EventsEmit(
                        "toast",
                        JSON.stringify(err),
                        faTimesCircle,
                        "red"
                    );
                });
        }
    }
    onMount(() => {
        IsTaskRun(title).then((res) => (working = res));
    });

    function checkPors() {
        return ports.map((t) => {
            for (var k in t) {
                t[k] = Number(t[k]);
            }
            return t;
        });
    }

    function save() {
        ports = checkPors();

        const task = { title, host, ports } as main.TaskConf;
        SaveTask(task);
    }
    function remove() {
        DelTask(title).then(() => {
            EventsEmit("toast", `删除了配置: ${title}`);
            ref.$destroy();
        });
    }
    let scale = 1.2;
</script>

<!-- class="m-auto"  -->
<Card bind:this={ref}>
    <div class="flex flex-end">
        <h1 class=" text-3xl font-bold grow justify-center">
            {title}
        </h1>
        {#if !working}
            <div class="flex-none">
                <Button on:click={remove} color="none">
                    <Icon data={faXmarkCircle} bind:scale />
                </Button>
            </div>
        {/if}
    </div>
    <Label class="space-y-2">
        <span>Host IP</span>
        <Input placeholder="Host IP" bind:value={host} />
    </Label>
    <Label>Ports</Label>
    <Table hoverable={true}>
        <TableHead>
            <TableHeadCell>Form</TableHeadCell>
            <TableHeadCell>To</TableHeadCell>
            <TableHeadCell>Action</TableHeadCell>
        </TableHead>
        <TableBody>
            {#each ports as port, i}
                <TableBodyRow>
                    <TableBodyCell>
                        <div contenteditable bind:textContent={port.from} />
                    </TableBodyCell>
                    <TableBodyCell>
                        <div contenteditable bind:textContent={port.to} />
                    </TableBodyCell>
                    <TableBodyCell>
                        <Button
                            on:click={() => DeleteRecord(i)}
                            color="none"
                            disabled={working}
                        >
                            <Icon data={faMinusCircle} bind:scale />
                        </Button>
                    </TableBodyCell>
                </TableBodyRow>
            {/each}
        </TableBody>
    </Table>
    <ButtonGroup class="m-auto">
        <Button
            class="space-y-2"
            color={working ? "red" : "green"}
            on:click={doWrok}
        >
            {#if working}
                <Icon data={faStop} bind:scale /> Stop
            {:else}
                <Icon data={faPlay} bind:scale /> Start
            {/if}
        </Button>
        <Button on:click={AddRecord} disabled={working} color="blue">
            <Icon data={faPlusCircle} bind:scale />新增
        </Button>
        <Button on:click={save} disabled={working} color="yellow">
            <Icon data={faSave} bind:scale />保存
        </Button>
    </ButtonGroup>
</Card>
